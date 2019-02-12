## Test Bench Information
prototype utilizes two scripts to satisfy scope
1. use aws-cli to get data about hosted zones and record sets
1. using curl and openssl to look at content served and certificates

testing done in a linux environment using the following setup:
- aws command line interface: `1.16.41`
- python: `2.7.10`
- curl: `7.29.0`
- openssl version `OpenSSL 1.0.2k-fips  26 Jan 2017`
