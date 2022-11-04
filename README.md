## How it works
This AWS CLI tool to read the SSO credentials cache and then makes `aws-sdk` calls to retrieve the temporary credentials for the relevant account/role you want.

It uses the standard AWS CLI configuration files, can trigger a SSO login session if needed and gives you an interactive command line interface to switch between the role and account you want. It will can also copy your chosen profile/credentials into the default profile for times where don't want/can't tell your application to use a specific profile.

## Prerequisites
A working installation of [AWS CLI v2](https://docs.aws.amazon.com/cli/latest/userguide/install-cliv2.html).

The scripts dependencies are defined in the `package.json` file. You can install these with: `yarn`

## Setting up
1. Install the AWS CLI v2 and [configure your profiles](https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-sso.html) as per the documentation. For example:

```ini
[profile dev-env]
region = eu-west-1
sso_start_url = https://yoursso.awsapps.com/start
sso_region = eu-west-1
sso_account_id = 123456654321
sso_role_name = DevOps

[profile prod-env]
region = eu-west-1
sso_start_url = https://yoursso.awsapps.com/start
sso_region = eu-west-1
sso_account_id = 543210012345
sso_role_name = DevOps
```

2. Run the AWS CLI tool *at least once* using one of the profiles you created so that the SSO cache is created.

```bash
  aws sso login --profile dev-env
```

## Usage

You can run `yarn awssso creds` passing it the name of the profile you want credentials for.

```bash
  $ yarn awssso creds dev-env
```

## Help

You can run

```bash
  $ yarn awssso --help
```

