The Apollo collector allows our support customers to easily submit bundles of information to the Graylog team. This way we can streamline support and dig deep into metrics without having to request more and more information via email or phone.

## Read before you run Apollo

* The machine you run Apollo from must be able to reach the REST APIs of all `graylog-server` instances in your Graylog cluster. If you are unsure we recommend not to run it from your workstation but from the machine your `graylog-server` master is running on.
* You must run Apollo with a Graylog admin user. This can either be the built-in administrator user or a custom user with the `administrator` ruleset applied. The bundle extraction will fail if you run it with a `reader` user.
* The information collected is usually not containing any sensitive information and never any messages you sent into Graylog. We will however be able to see stream and extractor names. Handling of the bundles falls under the support contract conditions and will thus never be shared and stored securely. You can look at the source code or unzip the generated bundle if you want to make sure.
* You only need to provide the REST API URL of one `graylog-server` instance. Apollo will auto-discover the other `graylog-server` instances in the cluster.
* Future version will allow an automatic transmission of the bundle to us.

### Using Apollo on Linux

* Download the binary: `curl -OL https://github.com/Graylog2/apollo/releases/download/0.2/apollo_linux_386`
* Make the binary executable: `chmod +x apollo_linux_386`
* Run the binary (replace username, password and hostname of `graylog-server` instance): `./apollo_linux_386 -user hans -password secret -url http://graylog.example.org:12900`
* Send us the generated `.ZIP` bundle file via email. (Located in same folder from where you executed Apollo. Called something like `graylog_apollo_bundle-2015-09-23T22-05-54.zip`)

### Using Apollo on OSX

* Download the binary: `curl -OL https://github.com/Graylog2/apollo/releases/download/0.2/apollo_osx_386`
* Make the binary executable: `chmod +x apollo_osx_386`
* Run the binary (replace username, password and hostname of `graylog-server` instance): `./apollo_osx_386 -user hans -password secret -url http://graylog.example.org:12900`
* Send us the generated `.ZIP` bundle file via email. (Located in same folder from where you executed Apollo. Called something like `graylog_apollo_bundle-2015-09-23T22-05-54.zip`)

### Using Apollo on Microsoft Windows

* Download the binary from the [Releases](https://github.com/Graylog2/apollo/releases).
* Run the binary (replace username, password and hostname of `graylog-server` instance): `c:\apollo_windows_386.exe -user hans -password secret -url http://graylog.example.org:12900`
* Send us the generated `.ZIP` bundle file via email. (Located in same folder from where you executed Apollo. Called something like `graylog_apollo_bundle-2015-09-23T22-05-54.zip`)

## FAQ

### I am getting a HTTP 404 and Apollo exits
Make sure to use a `graylog-server` REST API URL (Default port 12900) and not the URL of a `graylog-web-interface` as `-url` parameter.

### I am getting a HTTP 401 and Apollo exits
Make sure to use a user with administrator permissions in the `-user` and `-password` parameters of Apollo.

### What data is included in the bundle?
The information collected is usually not containing any sensitive information and never any messages stored in Graylog. We will however be able to see stream and extractor names. Handling of the bundles falls under the support contract conditions and will thus never be shared and stored securely. You can look at the source code or unzip the generated bundle if you want to make sure.

### Who will be able to see the bundles?
Only Graylog employees will be able to see data from the bundles.
