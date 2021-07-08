NovelAI Research Tool - `nrt`
=============================
A `golang` based client with:
* Minimum Viable Product implementation of a NovelAI service API client
  covering:
  - `/user/login` - to obtain authentication bearer tokens.
  - `/ai/generate` - to submit context and receive responses back from the AI
* Iterative testing based on JSON configuration files.  

Building
--------
You will need the `golang` language tools on your machine.

https://golang.org/doc/install

In this directory:
  * `go get -u`
  * `go build nrt.go`

This will produce a binary `nrt` file.

Setup
-----
The `nrt` tool uses environment variables to hold your NovelAI username and
password.  They are:
  * `NAI_USERNAME`
  * `NAI_PASSWORD`
You might add them to a `.login` file or to an `.env` file that you `source` any time you want to use the tool.

Running
-------
The `nrt` tool accepts a single filename as an argument, the `.json` file containing test parameters. They are more or
less self-explanatory, but I will highlight some specific ones:
  * `prompt_filename` - where to get the prompts, this is a `txt` file for easy editing of prompts without having to
escape like you would in JSON.
  * `output_filename` - where you want the JSON output from the generations to go.
  * `iterations` - how many times to run the test, effectively.
  * `generations` - how many times to take the output, concatenate, and re-feed back into the AI, like an user.
  * `parameters` - contains NovelAI configuration parameters according to the API's specifications.
An example `tests/need_help.json` can be used as a template, along with a `tests/need_help.txt` prompt file.
There is also an example `tests/need_help_output.json` file that contains an example of output.
    
**PLEASE NOTE:** There are hardcoded relative path assumptions and that the `nrt` tool should be run from this directory until I have a chance to clean it up.