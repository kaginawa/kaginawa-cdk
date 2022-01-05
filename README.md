# kaginawa-cdk

An AWS CDK (Cloud Development Kit) project that constructs the following building blocks required for the Kaginawa
Server.

- Certificate Manager (SSL/TLS certificate for the domain name)
- VPC
- EC2 instance (for the SSH server)
- DynamoDB tables
- ECS/Fargate cluster and service

## Parameters

Create `.env` file and fill parameters as following format.

```
OAUTH_TYPE=auth0  # or "google"
OAUTH_DOMAIN=example.auth0.com  # auth0 only
OAUTH_CLIENT_ID=xxxxx
OAUTH_CLIENT_SECRET=xxxxx
DOMAIN_NAME=kaginawa.example.com
KEYPAIR_NAME=kaginawa
NUM_OF_SSH_SERVERS=1
```

Manual operations:

- Create EC2 key pair with [Management Console](https://console.aws.amazon.com/ec2/v2/home#KeyPairs:) before deployment. 
- Add a DNS record (CNAME) for the domain verification during deployment process.

## CDK commands

* `cdk deploy`      deploy this stack to your default AWS account/region
* `cdk diff`        compare deployed stack with current state
* `cdk synth`       emits the synthesized CloudFormation template
* `go test`         run unit tests

## LICENSE

kaginawa-cdk licenced under the [BSD 3-Clause License](LICENSE).

## Author

[mikan](https://github.com/mikan)
