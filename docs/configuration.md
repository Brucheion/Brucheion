# Configuration

Brucheion is configured using a JSON-encoded configuration file, `config.json`, that resides next to the Brucheion executable.

```JSON
{
  "host": "http://localhost:7000",
  "port": ":7000",
  "gitHubKey": "20bitkey",
  "gitkHubSecret": "40bitsecret",
  "gitLabKey": "64bitkey",
  "gitLabSecret": "64bitsecret",
  "gitLabScope": "read_user",
  "maxAge": "43200",
  "userDB": "./users.db"
}
```

* `host`: The public address your Brucheion server will use.
* `port`: The port needs to be redefined for some functions to work.
* `gitHubKey`: The application key received from GitHub. This should be a 20 bit key, called 'Client ID' in the OAuth application settings.
* `githHubSecret`: The application secret received from GitHub. This should be a 40 bit key, called 'Client Secret' in the OAuth application settings.
* `gitLabSecret`: The application key received from GitLab. This should be a 64 bit key, called 'Application ID' in the application user settings.
* `gitLabSecret`: The application secret received from GitLab. This should be a 64 bit key, called 'Secret' in the application user settings.
* `gitLabScope`: "read_user" is necessary to properly set up the login via GitLab. Leave this line unaltered.
* `maxAge`: The time to live for the Brucheion session and its respective cookie in seconds. It may be set to a value that seems fitting for your scenarios. (A specific amount of days can be set multiplying 86400 by the amount of days. So for one day the line would be `"maxAge": "86400 * 1",`)
* `userDB`: The location where the user database will be saved. By default, it will be saved in the same folder the Brucheion executable resides. If you don't have a user database yet, one will be created with the first execution of Brucheion.
