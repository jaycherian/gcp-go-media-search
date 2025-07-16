#!/bin/bash
# 2-create-server.sh
# ==============================================================================
#  Interactive gcloud VM Creation Script
#
#  This script interactively asks for necessary inputs and then constructs
#  a `gcloud compute instances create` command based on your selections.
#  It's designed to be run in Google Cloud Shell.
#
#  How to use:
#  1. Save this script as a file (e.g., `create_vm.sh`).
#  2. Make it executable: `chmod +x create_vm.sh`
#  3. Run it: `./create_vm.sh`
# ==============================================================================

# --- Helper Functions for UI ---

# Prints a header for a section
print_header() {
    echo ""
    echo "======================================================================"
    echo "$1"
    echo "======================================================================"
}

# Prints an error message and exits
exit_with_error() {
    echo ""
    echo "❌ ERROR: $1" >&2
    exit 1
}

# --- Script Start ---

echo "Welcome to the Interactive VM Creation Script!"
echo "This will guide you through creating a gcloud command."

# --- 1. Get Project ID ---
print_header "Step 1: Project ID"
# Try to get the project from the current gcloud config
CURRENT_PROJECT=$(gcloud config get-value project 2>/dev/null)
if [[ -n "$CURRENT_PROJECT" ]]; then
    read -p "Enter your Project ID [default: $CURRENT_PROJECT]: " PROJECT_ID
    PROJECT_ID=${PROJECT_ID:-$CURRENT_PROJECT}
else
    read -p "Enter your Project ID: " PROJECT_ID
    if [[ -z "$PROJECT_ID" ]]; then
        exit_with_error "Project ID cannot be empty."
    fi
fi
# Set the project for the rest of the script's gcloud commands
gcloud config set project "$PROJECT_ID" || exit_with_error "Failed to set project to '$PROJECT_ID'. Please check if the project exists and you have permissions."
echo "✅ Project set to: $PROJECT_ID"


# --- 2. Get Instance Name ---
print_header "Step 2: Instance Name"
read -p "Enter a name for your new VM instance: " INSTANCE_NAME
if [[ -z "$INSTANCE_NAME" ]]; then
    exit_with_error "Instance name cannot be empty."
fi
# Set device name to be the same as the instance name
DEVICE_NAME=$INSTANCE_NAME
echo "✅ Instance Name: $INSTANCE_NAME"


# --- 3. Get Zone and Region ---
print_header "Step 3: Zone Selection"
echo "Fetching available zones... (this may take a moment)"
# Get a list of zones and let the user choose
ZONES=($(gcloud compute zones list --format="value(name)" | sort))
PS3="Please select a zone for your VM: "
select ZONE in "${ZONES[@]}"; do
    if [[ -n "$ZONE" ]]; then
        # Derive region from the zone (e.g., "us-central1-a" -> "us-central1")
        REGION=${ZONE%-*}
        echo "✅ Zone selected: $ZONE"
        echo "✅ Region automatically set to: $REGION"
        break
    else
        echo "Invalid selection. Please try again."
    fi
done


# --- 4. Get VPC Network ---
print_header "Step 4: VPC Network Selection"
echo "Fetching available VPC networks..."
# Get a list of VPC networks in the project
NETWORKS=($(gcloud compute networks list --project=$PROJECT_ID --format="value(name)"))

if [ ${#NETWORKS[@]} -eq 0 ]; then
    echo "No VPC networks found in project '$PROJECT_ID'."
    read -p "Would you like to create a new VPC network now? (y/N): " CREATE_VPC_CHOICE
    if [[ "$CREATE_VPC_CHOICE" == "y" || "$CREATE_VPC_CHOICE" == "Y" ]]; then
        read -p "Enter a name for the new VPC network: " VPC_NETWORK
        if [[ -z "$VPC_NETWORK" ]]; then
            exit_with_error "VPC network name cannot be empty."
        fi
        echo "Creating VPC network '$VPC_NETWORK' with custom subnet mode..."
        gcloud compute networks create "$VPC_NETWORK" --project="$PROJECT_ID" --subnet-mode=custom || exit_with_error "Failed to create VPC network '$VPC_NETWORK'."
        echo "✅ VPC Network '$VPC_NETWORK' created successfully."
    else
        exit_with_error "No VPC network available. Aborting script."
    fi
else
    PS3="Please select a VPC network: "
    select VPC_NETWORK in "${NETWORKS[@]}"; do
        if [[ -n "$VPC_NETWORK" ]]; then
            echo "✅ VPC Network selected: $VPC_NETWORK"
            break
        else
            echo "Invalid selection. Please try again."
        fi
    done
fi


# --- 5. Get Subnet ---
print_header "Step 5: Subnet Selection"
echo "Fetching available subnets for '$VPC_NETWORK' in region '$REGION'..."
# Get a list of subnets in the selected region and network
SUBNETS=($(gcloud compute networks subnets list --network=$VPC_NETWORK --regions=$REGION --format="value(name)"))

# Check if any subnets were found
if [ ${#SUBNETS[@]} -eq 0 ]; then
    echo "The selected VPC network '$VPC_NETWORK' does not have a subnet in the '$REGION' region."
    read -p "Would you like to create a new subnet in this region? (y/N): " CREATE_SUBNET_CHOICE
    if [[ "$CREATE_SUBNET_CHOICE" == "y" || "$CREATE_SUBNET_CHOICE" == "Y" ]]; then
        read -p "Enter a name for the new subnet: " SUBNET
        if [[ -z "$SUBNET" ]]; then
            exit_with_error "Subnet name cannot be empty."
        fi
        read -p "Enter a CIDR range for the new subnet [default: 10.1.2.0/24]: " CIDR_RANGE
        CIDR_RANGE=${CIDR_RANGE:-"10.1.2.0/24"}
        echo "Creating subnet '$SUBNET' with range '$CIDR_RANGE'..."
        gcloud compute networks subnets create "$SUBNET" --project="$PROJECT_ID" --network="$VPC_NETWORK" --region="$REGION" --range="$CIDR_RANGE" || exit_with_error "Failed to create subnet."
        echo "✅ Subnet '$SUBNET' created successfully."
    else
        exit_with_error "No subnet available in '$REGION' for VPC '$VPC_NETWORK'. Aborting script."
    fi
else
    PS3="Please select a subnet: "
    select SUBNET in "${SUBNETS[@]}"; do
        if [[ -n "$SUBNET" ]]; then
            echo "✅ Subnet selected: $SUBNET"
            break
        else
            echo "Invalid selection. Please try again."
        fi
    done
fi


# --- 6. Get Service Account ---
print_header "Step 6: Service Account"
echo "Finding the default Compute Engine service account..."
# Get the project number from the project ID
PROJECT_NUMBER=$(gcloud projects describe "$PROJECT_ID" --format="value(projectNumber)")
# Construct the default service account email
SERVICE_ACCOUNT="${PROJECT_NUMBER}-compute@developer.gserviceaccount.com"
echo "✅ Using default Compute Engine service account: $SERVICE_ACCOUNT"


# --- 7. Get Disk Resource Policy ---
print_header "Step 7: Disk Snapshot Policy"
echo "Fetching available disk resource policies in region '$REGION'..."
# Get a list of policies and let the user choose one
POLICIES=($(gcloud compute resource-policies list --regions=$REGION --format="value(name)"))
# Add an option for no policy
POLICIES+=("NONE")

PS3="Please select a disk snapshot policy (or 'NONE'): "
select DISK_POLICY in "${POLICIES[@]}"; do
    if [[ -n "$DISK_POLICY" ]]; then
        if [[ "$DISK_POLICY" == "NONE" ]]; then
             DISK_POLICY_FLAG=""
             echo "✅ No disk snapshot policy will be applied."
        else
            DISK_POLICY_FLAG="--disk-resource-policy=projects/$PROJECT_ID/regions/$REGION/resourcePolicies/$DISK_POLICY"
            echo "✅ Disk policy selected: $DISK_POLICY"
        fi
        break
    else
        echo "Invalid selection. Please try again."
    fi
done


# --- 8. Construct and Display the Final Command ---
print_header "Step 8: Final Command"
echo "Based on your selections, here is the generated gcloud command."
echo "Review it carefully before running."

# Base command construction. You can customize the static parts here.
# Note: The disk-resource-policy is handled by the $DISK_POLICY_FLAG variable, but needs to be added to the --create-disk flag
if [[ -n "$DISK_POLICY_FLAG" ]]; then
    CREATE_DISK_ARGS="auto-delete=yes,boot=yes,device-name=$DEVICE_NAME,image=projects/ubuntu-os-cloud/global/images/ubuntu-2004-focal-v20240110,mode=rw,size=100,type=pd-ssd,$DISK_POLICY_FLAG"
else
    CREATE_DISK_ARGS="auto-delete=yes,boot=yes,device-name=$DEVICE_NAME,image=projects/ubuntu-os-cloud/global/images/ubuntu-2004-focal-v20240110,mode=rw,size=100,type=pd-ssd"
fi


FINAL_COMMAND="gcloud compute instances create $INSTANCE_NAME \\
    --project=$PROJECT_ID \\
    --zone=$ZONE \\
    --machine-type=n2d-standard-16 \\
    --network-interface=network-tier=PREMIUM,nic-type=GVNIC,stack-type=IPV4_ONLY,subnet=$SUBNET \\
    --metadata=enable-osconfig=TRUE,enable-oslogin=true \\
    --maintenance-policy=MIGRATE \\
    --provisioning-model=STANDARD \\
    --service-account=$SERVICE_ACCOUNT \\
    --scopes=https://www.googleapis.com/auth/cloud-platform \\
    --create-disk=$CREATE_DISK_ARGS \\
    --shielded-secure-boot \\
    --shielded-vtpm \\
    --shielded-integrity-monitoring \\
    --reservation-affinity=any"

# Print the final command with syntax highlighting
echo ""
echo -e "\e[1;36m$FINAL_COMMAND\e[0m" # Cyan color for the command
echo ""

# --- Ask user to execute the command ---
read -p "Do you want to execute this command now? (y/N): " EXECUTE_CHOICE
if [[ "$EXECUTE_CHOICE" == "y" || "$EXECUTE_CHOICE" == "Y" ]]; then
    echo "Executing command..."
    # Execute the command and check if it was successful
    if eval "$FINAL_COMMAND"; then
        echo "✅ VM '$INSTANCE_NAME' created successfully."

        # --- NEW: Step 9: Firewall Configuration ---
        print_header "Step 9: Firewall Configuration"
        read -p "Do you want to allow internet traffic for SSH, HTTP, HTTPS, Ping, and Chrome Remote Desktop? (y/N): " CREATE_FW_CHOICE
        if [[ "$CREATE_FW_CHOICE" == "y" || "$CREATE_FW_CHOICE" == "Y" ]]; then
            FW_RULE_NAME="allow-common-internet-access"
            FW_TAG_NAME="allow-common-internet-access"

            # Check if the firewall rule already exists to avoid errors
            if ! gcloud compute firewall-rules describe "$FW_RULE_NAME" --project="$PROJECT_ID" &>/dev/null; then
                echo "Creating firewall rule '$FW_RULE_NAME'..."
                gcloud compute firewall-rules create "$FW_RULE_NAME" \
                    --project="$PROJECT_ID" \
                    --description="Allow incoming traffic for SSH, HTTP, HTTPS, Ping, and Chrome Remote Desktop" \
                    --direction=INGRESS \
                    --priority=1000 \
                    --network="$VPC_NETWORK" \
                    --action=ALLOW \
                    --rules=tcp:22,tcp:80,tcp:443,tcp:3389,tcp:4000,tcp:5173,icmp \
                    --source-ranges=0.0.0.0/0 \
                    --target-tags="$FW_TAG_NAME" || echo "⚠️  Could not create firewall rule. Please check permissions."
            else
                echo "✅ Firewall rule '$FW_RULE_NAME' already exists."
            fi

            echo "Applying tag '$FW_TAG_NAME' to VM '$INSTANCE_NAME'..."
            gcloud compute instances add-tags "$INSTANCE_NAME" \
                --project="$PROJECT_ID" \
                --zone="$ZONE" \
                --tags="$FW_TAG_NAME" || echo "⚠️  Could not apply tag to the VM."
            
            echo "✅ Firewall configuration complete."
        fi
        # --- End of Firewall Section ---
        
    else
      exit_with_error "VM creation failed. Please check the gcloud output for errors."
    fi
else
    echo "Command not executed. You can copy and paste it to run manually."
fi

echo ""
echo "Script finished."
