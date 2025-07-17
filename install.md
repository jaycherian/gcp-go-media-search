# Project Cairo \- Pilot Environment Setup

These instructions will walk through and explain the setup of a GCP environment for the Project Cairo pilot as well as installing the pilot application itself.

The Google Cloud Shell will be used for this setup so make sure you have access to it.

## Google Cloud Environment Setup

We will be using the Cloud Shell for this set up.

### Variables 

These are used in this text, change these values to whatever you want to use.

**Project ID:** `my-media-search-project-2`  
**VM Name:** `media-vm-2`  
**Zone:** `us-central1-b`  
**Release:** `release-0.0.3`   
**High Res Bucket:** `high-res-bucket-lexi-1`   
**Low Res Bucket:** `low-res-bucket-lexi-1`

### Create Project

Create a new project from the Google Cloud Console or from the Cloud Shell using these commands:

```
export PROJECT="my-media-search-project-2"
export ACCOUNT_ID=$(gcloud billing accounts list --format="value(ACCOUNT_ID)" --limit=1)

gcloud projects create $PROJECT
gcloud config set project $PROJECT
gcloud billing projects link $PROJECT --billing-account=$ACCOUNT_ID
```

### 

### Clone Repository

In the Cloud Shell, clone the [repository](https://github.com/jaycherian/gcp-go-media-search). We will be cloning to our home directory, but it can live anywhere you want:

```
cd ~
git clone https://github.com/jaycherian/gcp-go-media-search.git media-search
```

Now switch into the cloned folder and checkout the correct branch

```
cd media-search
git checkout muziris
```

### Run Scripts

We’ll now run scripts from the `deploy/scripts` folder to set up various things.

```
cd deploy/scripts
```

#### Enable APIs

Before enabling APIs you can list our which APIs are already enabled:

```
gcloud services list --filter="STATE:ENABLED" --format="value(NAME,STATE)"
```

This script will enable the Google Cloud APIs we’ll need for this app:

```
./1-enable-apis.sh
```

**NOTE**: Within this script we are creating the Vertex AI Service Agent which does not get created by default when enabling the API. We will also grant that service agent the roles it will need for the application.

#### Create and Setup the Server VM

First we’re going to create a VM to hold the server for our application.

```
./2-create-server.sh
```

Next, SSH to the new VM, replacing these values with your own.

```
gcloud compute ssh media-vm-2 --zone us-central1-b
```

Now that you’re logged into the VM, run this setup script:

```
bash -c "$(curl -fsSL https://raw.githubusercontent.com/jaycherian/gcp-go-media-search/refs/tags/release-0.0.3/deploy/scripts/3-setup-server.sh)"
```

Now we need to source our `.bashrc` file to update our PATH:

```
source ~/.bashrc
```

Now you can exit out of your ssh session:

```
exit
```

**NOTE**: This pilot requires external access to the VM running the application. If you’re running in a restricted environment such as **Argolis**, you’ll need to reset this org policy.

Run this command from the Cloud Shell:

```
gcloud org-policies reset compute.vmExternalIpAccess --project $GOOGLE_CLOUD_PROJECT
```

#### Create Service Accounts

Make sure you’re in the Cloud Shell and change dir back to the git repo:

```
cd ~/media-search/deploy/scripts
```

Next we’ll need to create a service account for our application, grant it some roles and update the default Compute service account to grant it more roles as well.

First create our application service account:

```
./4-setup-media-search-sa.sh
```

Now update the default Compute service account:

```
./5-setup-compute-sa.sh
```

### 

### Run Terraform

We’ll use terraform to provision cloud resources that the application requires.

Change into the terraform directory:

```
cd ../terraform
```

Make a copy of the `tfvars` file:

```
cp terraform.tfvars.example terraform.tfvars
```

Edit the `terraform.tfvars` file and populate the variables with the names you want to use:

* `high_res_bucket` \- The name of the storage bucket to hold high res video  
* `low_res_bucket` \- The name of the storage bucket to hold low res video  
* `project_id` \- You project ID

```
# Defining the bucket names for high resolution media. Please define a unique name as this bucket will be created in your project
high_res_bucket = "high-res-bucket-lexi-1"

# Defining the bucket name for low resolution media. Please define a unique name as this bucket will be created in your project.
low_res_bucket = "low-res-bucket-lexi-1"

# Specify the project to create infrastructure.
project_id     = "my-media-search-project-2"

```

Initialize and run terraform

```
terraform init
terraform apply
```

## 

## Media Search App Install

### Install the Application

First SSH to the server VM

```
gcloud compute ssh media-vm-2 --zone us-central1-b
```

Download the application and untar it into the home directory:

```
mkdir -p ~/media-search && curl -L https://github.com/jaycherian/gcp-go-media-search/archive/refs/tags/release-0.0.3.tar.gz | tar -xz --strip-components=1 -C ~/media-search
```

Now we need to set up our golang environment variables. Start by copying the template file: 

```
cp ~/media-search/backend/go/configs/example_.env.local.toml ~/media-search/backend/go/configs/.env.local.toml
```

Now edit the file: `~/media-search/backend/go/configs/.env.local.toml` and fill in these values: 

```
[application]
google_project_id = "my-media-search-project-2"
signer_service_account_email = "media-search-sa@my-media-search-project-2.iam.gserviceaccount.com"

[storage]
high_res_input_bucket = "high-res-bucket-lexi-1"
low_res_output_bucket = "low-res-bucket-lexi-1"
```

### Run the Application Servers

The application has two components that need to be run, the front end and the back end.

Open two terminal windows (or two tmux or screen sessions) and ssh into the server VM in each window using this command:

```
gcloud compute ssh media-vm-2 --zone us-central1-b
```

Run the frontend with this command (in one ssh session):

```
cd ~/media-search
./start_frontend.sh
```

Run the backend with this command (in the other ssh session):

```
cd ~/media-search
./start_backend.sh
```

### Open Media Search UI

Now we’re ready to access the media search application in a browser.

Get the external IP of the application VM with this command:

```
gcloud compute instances describe media-vm-2 --zone=us-central1-b  --format='get(networkInterfaces[0].accessConfigs[0].natIP)'
```

Assuming your IP is: 34.46.5.162, 

Browse to this URL: [http://](http://120.120.2.2:5173)34.46.5.162[:5173](http://120.120.2.2:5173)

You should see this UI:

![][image1]

[image1]: <data:image/png;base64,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>