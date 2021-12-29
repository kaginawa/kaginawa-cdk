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
OAUTH_DOMAIN=example.auth0.com
OAUTH_CLIENT_ID=xxxxx
OAUTH_CLIENT_SECRET=xxxxx
DOMAIN_NAME=kaginawa.example.com
```

We recommend to use [Auth0](https://auth0.com/) as OAuth2 provider.

You must add a DNS record (CNAME) for the domain verification during deployment process.

## CDK commands

* `cdk deploy`      deploy this stack to your default AWS account/region
* `cdk diff`        compare deployed stack with current state
* `cdk synth`       emits the synthesized CloudFormation template
* `go test`         run unit tests

## LICENSE

kaginawa-cdk licenced under the [BSD 3-Clause License](LICENSE).

## Author

[mikan](https://github.com/mikan)
