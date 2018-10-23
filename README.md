<img src="static/img/BrucheionLogo.png" alt="" width="500" height="500" align="middle">

Brucheion is a Virtual Research Environment (VRE) to create Linked Open Data (LOD) for historical languages and the research of historical objects.

## Using Brucheion

Parts of Brucheion are ready to be tested. Please note, that the VRE is still under development and potentially buggy.

## Install 

To Install Brucheion simply download the repository. Depending on the operating system you want to test Brucheion on, you may need to recompile the binary file. Compiling on linux for a 64 bit linux the Terminal you may use: `env GOOS=linux GOARCH=amd64 go build` See this overview explanation of [How To Build Go Executables for Multiple Platforms](https://www.digitalocean.com/community/tutorials/how-to-build-go-executables-for-multiple-platforms-on-ubuntu-16-04).


## Usage

If you simply want to try out Brucheion with the standard configuration you can start it using the terminal:

![Starting Brucheion using the terminal](static/img/tutorial/callFromTerminal.png)

As soon as Brucheion is running you may access it using your favorite browser. 

### Configuration

Brucheion is configured using a file called the config.json. 

![config.json](static/img/tutorial/jsonConfig.png)

JSON is a common data format used for asynchronous browser–server communication, that uses human-readable text to transmit data objects consisting of attribute–value pairs and array data types.<sup id="1">[1](#Wikipedia_JSON)</sup> Therefore it is essential to format config.json correctly.

```JSON
{
"host": "http://localhost:7000",
"port": ":7000",
"gitHubKey": "20bitkey",
"githHubSecret": "40bitsecret",
"gitLabKey": "64bitkey",
"gitLabSecret": "64bitsecret",
"gitLabScope": "read_user",
"maxAge": "43200",
"userDB": "./users.db"
}

//"googleKey": "",
//"googleSecret": "",
```

* The file content starts with { and ends with }. 
* Comments can be added __outside__ of the parenthesis using two forward slashes //
* Every entry name needs to be written inside parenthesis "" followed by colon and the value in parenthesis.
* Every, but the last line need to be separated with a comma
* "host" defines the address your Brucheion server will use
* "port" the port needs to be redefined for some functions to work
* "gitHubKey" defines the application key received from GitHub. This should be a 20 bit key, called 'Client ID' in the OAuth application settings.
* "githHubSecret" defines the application secret received from GitHub. This should be a 40 bit key, called 'Client Secret' in the OAuth application settings.
* "gitLabSecret" defines the application key received from GitLab. This should be a 64 bit key, called 'Application ID' in the applications user settings.
* "gitLabSecret" defines the application secret received from GitLab. This should be a 64 bit key, called 'Secret' in the applications user settings.
* "gitLabScope": "read_user" is necessary to properly set up the login via GitLab. Just leave this line unaltered.
* "maxAge" defines the time to live for the Brucheion session and its respective cookie in seconds. It may be set to a value that seems fitting for your scenarios. (A specific amount of days can be set multiplying 86400 by the amount of days. So for one day the line would be `"maxAge": "86400 * 1",`)
* "userDB" defines the location where the user database will be saved. By default it will be saved in the same folder the Brucheion executable resides. If you don't have a user database yet, one will be created with the first execution of Brucheion.
* Google may be added as a login provider later.

### Login

To login you need a GitHub or a GitLab account. 

1. If the server is running navigate to /login/ (with the standard configuration navigate to http://localhost:7000/login/)

![loginPage](static/img/tutorial/loginBlank.png)

2. Enter a user name of your liking. For this tutorial it'll be Adri.

![loginAdri](static/img/tutorial/loginAdri.png)

3. Choose the login provider that you want to use to authenticate and click on login. If the user is new you should see the following confirmation message:

![loginNewUserGitHubsuccess](static/img/tutorial/newUserGHsuccess.png)

4. To choose another provider simply use the drop-down menu:

![loginChooseProvider](static/img/tutorial/loginChooseProvider.png)

5. But please always use the same user name with the same login provider. For example: if user Adri tries to login using his GitLab account, Brucheion will show an error message:

![loginUsernameInUse](static/img/tutorial/loginUsernameInUse.png)

6. Likewise a new user can not be registered with an GitHub or GiLab account that is already in use.

### View

<b id="Wikipedia_JSON">1</b> https://en.wikipedia.org/wiki/JSON [↩](#1)


