# Auto-Sync Evidence to Hyperproof

## 1. Objective and Value
The purpose of this document is to detail the architecture and workflow for automatically syncing compliance evidence into [Hyperproof](https://hyperproof.io/). This process automates the "last mile" of the compliance journey: delivering collected, enriched, and verified evidence directly into the organisation's GRC (Governance, Risk, and Compliance) platform.

---

### **Business Value**
Implementing this workflow closes the loop between technical operations and compliance auditing, achieving:

* Continuous Compliance: Transforms evidence collection from a periodic, manual scramble into a continuous, automated flow.
* Audit Readiness: Ensures evidence is instantly available to auditors and stakeholders within [Hyperproof](https://hyperproof.io/).
* End-to-End Automation: Fully automates the pipeline from code check-in (or system event) to auditor review.

---

## 2. Technical Architecture & Workflow
The automation pipeline uses an event-driven architecture hosted on [AWS](https://docs.aws.amazon.com/) to bridge [Complybeacon](https://github.com/complytime/complybeacon) and [Hyperproof](https://hyperproof.io/).



### **The Step-by-Step Workflow**

| Step | Component | Action | Details |
| :--- | :--- | :--- | :--- |
| Export | Complybeacon | Output | Complybeacon completes evidence collection and exports the finalized logs. |
| Ingestion | AWS S3 | Secure Storage | The evidence logs are deposited into the designated S3 Bucket. |
| Trigger | S3 Event | Event-Driven | The creation of a new object in S3 automatically triggers the linked AWS Lambda Function. |
| Processing | AWS Lambda | Transformation/Push | The function executes a Python script that retrieves the Hyperproof secrets from AWS SSM, authenticates via the Hyperproof API, and pushes the evidence data. |
| Verification | AWS / Hyperproof | Validation | Inspect CloudWatch Logs for successful execution. Then, check Hyperproof to verify the evidence appears in the expected location. |

---

## 3. Preparation & Prerequisites
Before configuring the automation, the following components and credentials must be provisioned.

### **3.1 Hyperproof Configuration**

1.  **Provision API Credentials:** Create an API client within Hyperproof to allow external access.
    * *Path:* `Administrator -> Setting -> API Client`
2.  **Record Credentials:** Securely note the `CLIENT_ID` and `CLIENT_SECRET`.

### **3.2 AWS Infrastructure Setup**

#### **A. IAM & [S3 Bucket](https://docs.aws.amazon.com/s3/?icmpid=docs_homepage_featuredsvcs) (Storage)**

1.  Create S3 Bucket: Provision a new AWS S3 bucket for evidence ingestion. Note the Bucket Name.
2.  [Create IAM Policy](https://docs.hyperproof.io/cm/en/integrations/hp-amazon-s3): Create an IAM Policy granting write access to this specific S3 bucket (for Complybeacon).

    _Example Policy snippet_
    ```
        {
            "Version": "2012-10-17",
            "Statement": [
                {
                    "Sid": "VisualEditor0",
                    "Effect": "Allow",
                    "Action": [
                        "s3:GetObject"
                    ],
                    "Resource": [
                        "arn:aws:s3:::sw-s3-hyperproof/*", # Update the S3 bucket name
                        "arn:aws:s3:::sw-s3-hyperproof" # Update the S3 bucket name
                    ]
                },
                {
                    "Sid": "VisualEditor1",
                    "Effect": "Allow",
                    "Action": [
                        "s3:ListAllMyBuckets",
                        "s3:ListBucket"
                    ],
                    "Resource": "*"
                },
                {
                    "Sid": "VisualEditor2",
                    "Effect": "Allow",
                    "Action": "s3:PutObject",
                    "Resource": "arn:aws:s3:::sw-s3-hyperproof/*" # Update the S3 bucket name
                }
            ]
        }
    ```
3.  [Create IAM User](https://docs.hyperproof.io/cm/en/integrations/hp-amazon-s3): Create an IAM User (for Complybeacon), attach the policy, and generate the `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY`.

#### **B. [Systems Manager](https://docs.aws.amazon.com/systems-manager/?icmpid=docs_homepage_mgmtgov) (Secrets Management)**

Create new **`SecureString`** parameters in the AWS Systems Manager (SSM) Parameter Store to securely hold the Hyperproof credentials.

* `/hyperproof/CLIENT_ID`
* `/hyperproof/CLIENT_SECRET`

#### **C. [Lambda Function](https://docs.aws.amazon.com/lambda/?icmpid=docs_homepage_featuredsvcs)**

1.  **Create Function:** Initialise a new AWS Lambda function (e.g., using Python runtime).
2.  **Configure Triggers:** Add an S3 trigger linking it to the bucket from step **3.2 A**, configured to fire on `ObjectCreated` events.
3.  **Configure IAM Execution Role:**
    * Attach the managed policy `AmazonS3ReadOnlyAccess` (to allow Lambda to read the evidence logs).
    * Create and attach an inline policy granting `ssm:GetParameter` permission to read the specific SSM parameters (`/hyperproof/CLIENT_ID`, `/hyperproof/CLIENT_SECRET`).

    _Example Policy snippet_
    ```
        {
            "Version": "2012-10-17",
            "Statement": [
                {
                    "Sid": "AllowReadingSSMParameters",
                    "Effect": "Allow",
                    "Action": [
                        "ssm:GetParameter"
                    ],
                    "Resource": [
                        "arn:aws:ssm:*:*:parameter/hyperproof/CLIENT_ID", # Need to update
                        "arn:aws:ssm:*:*:parameter/hyperproof/CLIENT_SECRET" # Need to update
                    ]
                },
                {
                    "Sid": "AllowDecryptingSecrets",
                    "Effect": "Allow",
                    "Action": [
                        "kms:Decrypt"
                    ],
                    "Resource": "arn:aws:kms:*:*:alias/aws/ssm"
                }
            ]
        }
    ```
4.  **Dependencies & Layers:** Create and attach a Lambda Layer containing the necessary Python libraries (e.g., `requests`, `boto3`) for making API requests and interacting with AWS services.
5.  **Set Environment Variables:** Configure the following (for the Python script to use):
    * `SSM_CLIENT_ID_PATH`: `/hyperproof/CLIENT_ID`
    * `SSM_SECRET_PATH`: `/hyperproof/CLIENT_SECRET`
    * `CONTROL_ID` (Optional): The Hyperproof Control ID to which the evidence should be mapped by default.
6.  **Deploy Code:** Deploy the actual sync code (which reads S3, retrieves secrets from SSM, and calls the Hyperproof API) into the Lambda Function editor.

---

## 4. Execution
Once all prerequisites are complete, the pipeline is activated automatically:

1.  The Complybeacon exports the evidence log.
2.  The evidence log is written to the configured S3 bucket.
3.  The S3 write event immediately triggers the Lambda function.
4.  The Lambda function executes, pushing the evidence log to Hyperproof.
