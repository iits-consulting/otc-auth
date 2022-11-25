# OTC-Auth
Open Source CLI for the Authorization with the Open Telekom Cloud written in go.

With this CLI you can log in to the OTC through its Identity Access Manager (IAM) or through an external Identity Provider (IdP) in order to get an unscoped token. The allowed protocols for IdP login are SAML and OIDC. When logging in directly with Telekom's IAM it is also possible to use Multi-Factor Authentication (MFA) in the process.

After you have retrieved an unscoped token, you can use it to get a list of the clusters in a project from the Cloud Container Engine (CCE) and also get the remote kube config file and merge with your local file.

This tool can also be used to manage (create) a pair of Access Key/ Secret Key in order to make requests more secure.

## Login
Use the `login` command to retrieve an unscoped token either by logging in directly with the Service Provider or through an IdP. You can see the help page by entering `login --help` or `login -h`.

### Service Provider Login (IAM)
To log in directly with the Service Provider's IAM, you will have to supply the domain name you're attempting to log in to (usually starting with "OTC-EU", following the region and a longer identifier), your username and password. Please note that the argument `--os-auth-type` must be set to `iam`. 

`login --os-auth-type iam --os-domain-name <domain_name> --os-username <username> --os-password <password>`

Alternatively, it is possible to use MFA if that's desired and/or required. In this case both arguments `--os-user-id` and `--otp` are required. The user id can be obtained in the "My Credentials" page on the OTC. 

`login --os-auth-type iam --os-domain-name <domain_name> --os-user-id <user_id> --os-username <username> --os-password <password> --otp <6_digit_token>`

The OTP Token is 6-digit long and refreshes every 30 seconds. For more information on MFA please refer to the [OTC's documentation](https://docs.otc.t-systems.com/en-us/usermanual/iam/iam_10_0002.html).

### Identity Provider Login (IdP)
To log in through an IdP, you will have to supply its name and url, besides the usual authorization type, username and password information. The domain name is not mandatory in this case and the argument `--os-auth-type` must be set to `idp`.

`login --os-auth-type idp --os-auth-idp-name <idp_name> --os-idp-url <idp_url> --os-username <username> --os-password <password>`

Alternatively, you can pass the argument `--os-protocol` to set the required login method. The allowed methods are `saml` (default if the argument is not provided) or `oidc`.

`login --os-auth-type idp --os-auth-idp-name <idp_name> --os-idp-url <idp_url> --os-protocol oidc --os-username <username> --os-password <password>`

At the moment, only `saml` login is supported. Also, MFA is not supported on IdP login at the moment.

## Cloud Container Engine
Use the `cce` command to retrieve a list of available clusters in your project and/or get the remote kube configuration file. You can see the help page by entering `cce --help` or `cce -h`.

To retrieve a list of clusters for a project use the following command: 

`cce list --project <project_name>`

To retrieve the remote kube configuration file (and merge to your local one) use the following command:

`cce get-kube-config --project <project_name> --cluster <cluster_name>`

Alternatively you can pass the argument `--days-valid` to set the period of days the configuration will be valid, the default is 7 days.
