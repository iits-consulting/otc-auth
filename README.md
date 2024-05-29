# OTC-Auth

Open Source CLI for the Authorization with the Open Telekom Cloud.

[![License: GPLv3](https://img.shields.io/badge/GitHub-GPL--3.0-informational)](https://github.com/iits-consulting/otc-auth/blob/main/LICENSE)
![Build](https://github.com/iits-consulting/otc-auth/workflows/Build/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/iits-consulting/otc-auth)](https://goreportcard.com/report/github.com/iits-consulting/otc-auth)
![CodeQL](https://github.com/iits-consulting/otc-auth/workflows/CodeQL/badge.svg)
![ViewCount](https://views.whatilearened.today/views/github/iits-consulting/otc-auth.svg)

With this CLI you can log in to the OTC through its Identity Access Manager (IAM) or through an external Identity
Provider (IdP) in order to get an unscoped token. The allowed protocols for IdP login are SAML and OIDC. When logging in
directly with Telekom's IAM it is also possible to use Multi-Factor Authentication (MFA) in the process.

After you have retrieved an unscoped token, you can use it to get a list of the clusters in a project from the Cloud
Container Engine (CCE) and also get the remote kube config file and merge with your local file.

This tool can also be used to manage (create) a pair of Access Key/ Secret Key in order to make requests more secure.

## Table of contents

* [Demo](#demo)
* [Install](#install)
    * [Linux/BSD/Unix](#linuxbsdunix)
    * [Mac](#macos)
    * [Windows/Other](#windowsother)
* Usage
    * [Login](#login)
        * [Service Provider Login (IAM)](#service-provider-login-iam)
        * [Identity Provider Login (IdP)](#identity-provider-login-idp)
            * [External IdP and SAML](#external-idp-and-saml)
            * [External IdP and OIDC](#external-idp-and-oidc)
            * [Service Account via external IdP and OIDC](#service-account-via-external-idp-and-oidc)
        * [OIDC Scopes](#oidc-scopes)
        * [Remove Login](#remove-login)
    * [List Projects](#list-projects)
    * [Cloud Container Engine](#cloud-container-engine)
    * [Manage Access Key and Secret Key Pair](#manage-access-key-and-secret-key-pair)
    * [Openstack Integration](#openstack-integration)
    * [Environment Variables](#environment-variables)
    * [Auto-Completions](#auto-completions)
    * [Debugging](#debugging)

## Demo

https://user-images.githubusercontent.com/19291722/208880256-b0da924e-254e-4bc4-b9ee-396c43234a5b.mp4

## Install

### Linux/BSD/Unix

We have repos set up
for [Arch](https://aur.archlinux.org/packages/otc-auth), [Debian](https://github.com/iits-consulting/ppa), [Fedora](https://github.com/iits-consulting/rpm-repo)
and [Alpine](https://github.com/iits-consulting/apk-repo).

Alternatively, you can download and use the binaries from our [releases page](https://github.com/iits-consulting/otc-auth/releases). Remember to add it to your PATH! Replace {OTC_AUTH_VERSION} and {YOUR_PLATFORM} in the example below with the version you want to download and the platform you want to download it for.
```shell
curl -OL https://github.com/iits-consulting/otc-auth/releases/download/{OTC_AUTH_VERSION}/otc-auth_{YOUR_PLATFORM}.tar.gz
tar -xf otc-auth_{YOUR_PLATFORM}.tar.gz
sudo mv otc-auth /usr/local/bin/otc-auth                                                                                                                                                                                                                                                                            1
```

### MacOS
We recommend using our [brew](https://brew.sh) tap to install otc-auth. 

```shell
brew tap iits-consulting/homebrew-tap
brew install otc-auth
```

If you don't want to use brew, feel free to download the binary for your system from the [releases page](https://github.com/iits-consulting/otc-auth/releases). Remember to add it to your PATH!

### Windows/Other

Download the binary for your system from the [releases page](https://github.com/iits-consulting/otc-auth/releases).
Unpack the binary, add it to your PATH and you are good to go!

## Login

Use the `login` command to retrieve an unscoped token either by logging in directly with the Service Provider or through
an IdP. You can see the help page by entering `login --help` or `login -h`. There are three log in
options (`iam`, `idp-saml`, and `idp-oidc`) and one of them must be provided.

### Service Provider Login (IAM)

To log in directly with the Open Telekom Cloud's IAM, you will have to supply the domain name you're attempting to log
in to (usually starting with "OTC-EU", following the region and a longer identifier), your username and password.

```bash
otc-auth login iam --os-username <username> --os-password <password> --os-domain-name <domain_name> --region <region>
```

Additionally, it is possible to use MFA if needed. In this case, both
arguments `--os-user-domain-id` and `--totp`, are required (with `--os-user-domain-id` replacing `--os-username`). 
The user id can be obtained in the "My Credentials" page on the OTC.

```bash
otc-auth login iam --os-password <password> --os-domain-name <domain_name> --os-user-domain-id <user_domain_id> --totp <6_digit_token> --region <region>
```

The OTP Token is 6-digits long and refreshes every 30 seconds. For more information on MFA please refer to
the [OTC's documentation](https://docs.otc.t-systems.com/en-us/usermanual/iam/iam_10_0002.html).

### Identity Provider Login (IdP)

You can log in with an external IdP using either the `saml` or the `oidc` protocols. In both cases you will need to
specify the authorization URL, the name of the Identity Provider (as set on the OTC), as well as username and password
for the SAML login and client id (and optionally client secret) for the OIDC login flow.

#### External IdP and SAML

The SAML login flow is SP initiated and requires you to send username and password to the SP. The SP then authorizes you
with the configured IdP and returns either an unscoped token or an error, if the user is not allowed to log in.

```bash
otc-auth login idp-saml --os-username <username> --os-password <password> --idp-name <idp_name> --idp-url <authorization_url> --os-domain-name <os_domain_name> --region <region>
```

At the moment, no MFA is supported for this login flow.

#### External IdP and OIDC

The OIDC login flow is user initiated and will open a browser window with the IdP's authorization URL for the user to
log in as desired. This flow does support MFA (this requires it to be configured on the IdP). After being successfully
authenticated with the IdP, the SP will be contacted with the corresponding credentials and will return either an
unscoped token or an error, if the user is not allowed to log in.

```bash
otc-auth login idp-oidc --idp-name <idp_name> --idp-url <authorization_url> --client-id <client_id> --os-domain-name <os_domain_name> --region <region> [--client-secret <client_secret>]
```

The argument `--client-id` is required, but the argument `--client-secret` is only needed if configured on the IdP.

#### Service Account via external IdP and OIDC

If you have set up your IdP to provide service accounts then you can utilize service account with `otc-auth` too. Make
also sure that the IdP is correctly configured in the OTC Identity and Access Management. Then run the `otc-auth` as
follows:

```bash
otc-auth login idp-oidc \
    --idp-name NameOfClientInIdp \
    --idp-url IdpAuthUrl \
    --os-domain-name YourDomainName \
    --region YourRegion \
    --client-id NameOfIdpInOtcIam \
    --client-secret ClientSecretForTheClientInIdp \
    --service-account
```

### OIDC Scopes

The OIDC scopes can be configured if required. To do so simply provide one of the following two when logging in
with `idp-oidc`:

- provide the flag `--oidc-scopes pleasePut,HereAll,YourScopes,WhichYouNeed`
- provide the environment variable `export OIDC_SCOPES="pleasePut,HereAll,YourScopes,WhichYouNeed"`

The default value is `openid,profile,roles,name,groups,email`

### Remove Login

Clouds are differentiated by their identifier `--os-domain-name`. To delete a cloud, use the `remove` command.

```bash
otc-auth login remove --os-domain-name <os_domain_name> --region <region>
```

## List Projects

It is possible to get a list of all projects in the current cloud. For that, use the following command.

```bash
otc-auth projects list
```

## Cloud Container Engine

Use the `cce` command to retrieve a list of available clusters in your project and/or get the remote kube configuration
file. You can see the help page by entering `cce --help` or `cce -h`.

To retrieve a list of clusters for a project use the following command. The project name will be checked against the
ones in the cloud at the moment of the request.
If the desired project isn't found, you will receive an error message.

```bash
otc-auth cce list --os-domain-name <os_domain_name> --region <region> --os-project-name <project_name>
```

To retrieve the remote kube configuration file (and merge it to your local one) use the following command:

```bash
otc-auth cce get-kube-config --os-domain-name <os_domain_name> --region <region> --os-project-name <project_name> --cluster <cluster_name>
```

You could also pass the `--days-valid` argument to set the period of days the configuration will be valid for, with the
default being 7 days. The `-s` or `--server` argument could also be used to override the *server* attribute in the
config generated.

## Manage Access Key and Secret Key Pair

You can use the OTC-Auth tool to download permanent AK/SK pairs directly from the OTC. A file called "ak-sk-env.sh" will
be created in the current directory. The file contains four environment variables.

```bash
otc-auth access-token create --os-domain-name <os_domain_name> --region <region>
```

If a temporary AK/SK pair is needed instead, use the following command:

```bash
otc-auth temp-access-token create --os-domain-name <os_domain_name> --region <region> -t <lifetime in seconds>
```

This will generate a temporary AK/SK pair (valid for 15m by default, if the `-t` argument is not given), saved to "
ak-sk-env.sh".
The file will contain five environment variables.

The "ak-sk-env.sh" file must then be `source`-ed before you can start using the environment variables.

## Openstack Integration

The OTC-Auth tool is able to generate the clouds.yaml config file for openstack. With this file it is possible to
reuse the clouds.yaml with terraform.

If you execute this command

```bash
otc-auth openstack config-create
```

It will create a cloud config for every project which you have access to and generate a scoped token. After that it
overrides
the clouds.yaml (by default: ~/.config/openstack/clouds.yaml) file.

## Environment Variables

The OTC-Auth tool also provides environment variables for all the required arguments. For the sake of compatibility,
they are aligned with the Open Stack environment variables (starting with OS).

| Environment Variable  | Argument                  | Short | Description                                   |
|-----------------------|---------------------------|:-----:|-----------------------------------------------|
| CLIENT_ID             | `--client-id`             |  `c`  | Client id as configured on the IdP            |
| CLIENT_SECRET         | `--client-secret`         |  `s`  | Client secret as configured on the IdP        |
| CLUSTER_NAME          | `--cluster`               |  `c`  | Cluster name on the OTC                       |
| OS_DOMAIN_NAME        | `--os-domain-name`        |  `d`  | Domain Name from OTC Tenant                   |
| REGION                | `--region`                |  `r`  | Region code for the cloud (eu-de for example) |
| OS_PASSWORD           | `--os-password`           |  `p`  | Password (iam or idp)                         |
| OS_PROJECT_NAME       | `--os-project-name`       |  `p`  | Project name on the OTC                       |
| OS_USER_DOMAIN_ID     | `--os-user-domain-id`     |  `i`  | User id from OTC Tenant                       |
| OS_USERNAME           | `--os-username`           |  `u`  | Username (iam or idp)                         |
| IDP_NAME              | `--idp-name`              |  `i`  | Identity Provider name (as configured on OTC) |
| IDP_URL               | `--idp-url`               |  N/A  | Authorization endpoint on the IDP             |
| SKIP_TLS_VERIFICATION | `--skip-tls-verification` |  N/A  | Skips TLS Verification                        |

## Auto-Completions

You install the auto completions for your shell by running. Please follow the instructions by
running `otc-auth completion --help` in your terminal.

## Debugging

Is something not working the way you've expected? otc-auth uses [glog](https://pkg.go.dev/github.com/golang/glog) for logging with all info output at log-level 1.
In the following example, we'd like to have the logs from the OIDC login command be saved to our current directory:

```bash
otc-auth login idp-oidc -v 1 --log_dir .
```

We could also just print the logs to stderr instead of writing them to a file:

```bash
otc-auth login idp-oidc -v 1 --logtostderr=true
```

The more advanced logging features (like logging to both a file and stderr, emitting a stack trace at a specific line, buffering log messages and more) 
are described in the [glog documentation](https://pkg.go.dev/github.com/golang/glog#pkg-overview).
