// Copyright 2024 Google, LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package cloud provides components for interacting with Google Cloud services.
// This file defines a generic, reusable Pub/Sub message listener. The core idea
// is to abstract the complexity of receiving messages from a Pub/Sub subscription
// and to delegate the actual message processing to a "Command". This promotes
// separation of concerns, making the code cleaner and more modular.
//
// Logic Flow:
//  1. An instance of PubSubListener is created with a client and a subscription ID.
//  2. A "Command" (a piece of business logic) is attached to this listener.
//  3. The `Listen` method is called, which starts an asynchronous background process (a goroutine).
//  4. This goroutine continuously waits for new messages from the specified subscription.
//  5. When a message arrives, it's passed to the attached Command for processing.
//  6. The message is "acknowledged" (Ack'd) only if the Command completes successfully,
//     ensuring reliable, at-least-once message processing.
//  7. The entire process is instrumented with OpenTelemetry for tracing and monitoring.
//
// Structs:
//   - PubSubListener: Manages the connection to a Pub/Sub subscription and holds
//     the command that will process incoming messages.
//
// Functions:
//   - NewPubSubListener: Constructor for creating a new PubSubListener.
//   - SetCommand: Attaches a processing command to the listener.
//   - Listen: Starts the background process to receive and handle messages.
package cloud

import (
	"context"
	"log"

	"cloud.google.com/go/pubsub"
	"github.com/jaycherian/gcp-go-media-search/internal/core/cor"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// PubSubListener is a struct that encapsulates the components needed to listen
// to a specific Google Cloud Pub/Sub subscription. It acts as a wrapper that
// connects a subscription to a processing command. Since listeners have a
// life-cycle independent of individual API requests, they are considered a
// core "cloud" component.
type PubSubListener struct {
	client       *pubsub.Client       // The client for interacting with the Pub/Sub service.
	subscription *pubsub.Subscription // The specific subscription this listener will pull messages from.
	command      cor.Command          // The command to execute for each message received. This is part of the Chain of Responsibility (CoR) pattern.
}

// NewPubSubListener is the constructor for creating a PubSubListener. It initializes
// the listener with a Pub/Sub client, the ID of the subscription to listen to, and
// the command that will process the messages.
//
// Inputs:
//   - pubsubClient: An authenticated *pubsub.Client for connecting to the service.
//   - subscriptionID: The string ID of the subscription (e.g., "my-subscription").
//   - command: A cor.Command that defines the business logic to execute on each message.
//
// Outputs:
//   - *PubSubListener: A pointer to the newly created and configured listener.
//   - error: An error if the listener could not be created (though in this implementation, it always returns nil).
func NewPubSubListener(
	pubsubClient *pubsub.Client,
	subscriptionID string,
	command cor.Command,
) (cmd *PubSubListener, err error) {
	// Get a reference to the subscription object from the Pub/Sub client using its ID.
	sub := pubsubClient.Subscription(subscriptionID)

	// Create a new PubSubListener instance, populating its fields.
	cmd = &PubSubListener{
		client:       pubsubClient,
		subscription: sub,
		command:      command,
	}
	return cmd, nil
}

// SetCommand is a setter method that attaches a command to the listener.
// This is useful for scenarios where the listener is created before the full
// processing chain (the command) is assembled. It ensures that a command is not
// accidentally overwritten.
//
// Inputs:
//   - command: The cor.Command to be executed when a message is received.
func (m *PubSubListener) SetCommand(command cor.Command) {
	// Only set the command if it hasn't been set already. This prevents
	// accidental overwrites and ensures the initial configuration is respected.
	if m.command == nil {
		m.command = command
	}
}

// Listen starts the asynchronous message receiving process. It runs in a separate
// goroutine so it doesn't block the main application thread. This allows the server
// to continue handling other tasks (like API requests) while listening for messages
// in the background.
//
// Inputs:
//   - ctx: A context.Context that controls the lifecycle of the listener. If this
//     context is canceled (e.g., during graceful shutdown), the message receiving will stop.
func (m *PubSubListener) Listen(ctx context.Context) {
	log.Printf("listening: %s", m.subscription)

	// Launch a new goroutine for the background work. This is the Go way of
	// handling concurrent, non-blocking operations.
	go func() {
		// Create a tracer for this specific listener. The tracer is used to create "spans"
		// which are units of work in a distributed trace, helping to monitor and debug.
		tracer := otel.Tracer("message-listener")

		// The subscription.Receive method blocks and waits for messages. It takes a
		// callback function that will be executed for each message that arrives.
		err := m.subscription.Receive(ctx, func(_ context.Context, msg *pubsub.Message) {
			// Start a new span for the processing of this specific message. This allows
			// us to trace the journey of a single message through the system.
			spanCtx, span := tracer.Start(ctx, "receive-message")
			span.SetName("receive-message")
			// Attach the message data as an attribute to the span for better traceability.
			span.SetAttributes(attribute.String("msg", string(msg.Data)))
			log.Println("received message")

			// Create a new context for the Chain of Responsibility (CoR). This context
			// will carry data through the different steps of the processing command.
			chainCtx := cor.NewBaseContext()
			chainCtx.SetContext(spanCtx)              // Pass the tracing span's context into the chain.
			chainCtx.Add(cor.CtxIn, string(msg.Data)) // Add the message data as the initial input.

			// Execute the command attached to the listener, passing the chain's context.
			m.command.Execute(chainCtx)

			// Check if the command (and its entire chain) executed without any errors.
			if !chainCtx.HasErrors() {
				// If successful, set the span's status to Ok.
				span.SetStatus(codes.Ok, "success")
				// Acknowledge the message. This tells Pub/Sub that the message has been
				// successfully processed and can be deleted from the subscription.
				msg.Ack()
			} else {
				// If there were errors, set the span's status to Error.
				span.SetStatus(codes.Error, "failed")
				// Log each error that occurred during the chain's execution.
				for _, e := range chainCtx.GetErrors() {
					log.Printf("error executing chain: %v", e)
				}
				// By *not* calling msg.Ack() or msg.Nack(), we allow the message to
				// be redelivered after its acknowledgement deadline expires,
				// following the subscription's retry policy.
			}

			// End the span to mark the completion of this unit of work.
			span.End()
		})

		// If the Receive call exits (e.g., because the context was canceled),
		// check if there was an error and log it.
		if err != nil {
			log.Printf("error receiving data: %v", err)
		}
	}()
}
