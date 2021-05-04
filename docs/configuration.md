# Configuration

Brucheion is configured using a JSON-encoded configuration file (`config.json`) that resides next to the Brucheion executable.

```json
{
  "host": "http://localhost:7000",
  "port": ":7000",
  "maxAge": 86400,
  "userDB": "./users.db",
  "orthographyNormalisationFilenames": {
    "san": "SanskritOrthography.json",
    "sans": "SanskritOrthography.json"
  },
  "useNormalization": true

}
```

* `host`: The public address your Brucheion server will use.
* `port`: The port needs to be redefined for some functions to work.
* `maxAge`: The time to live for the Brucheion session and its respective cookie in seconds. It may be set to a value that seems fitting for your scenarios. (A specific amount of days can be set multiplying 86400 by the amount of days. So for one day the line would be `"maxAge": "86400 * 1",`).
* `orthographyNormalisationFilenames`: Filenames for orthography settings.
* `userDB`: The location where the user database will be saved. By default, it will be saved in the same folder the Brucheion executable resides. If you don't have a user database yet, one will be created with the first execution of Brucheion.

