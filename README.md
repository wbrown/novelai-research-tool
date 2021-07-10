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

* **Windows:** Download and install from here: https://golang.org/doc/install
* **Mac:** If do not use `brew`, you could download from
  https://golang.org/doc/install; otherwise `brew install go`
* **Linux:** Use your package maager to install `go` or `golang`. If you're
  using Linux, I presume you know how to do this.

Get a copy of the source code either by:
* Downloading the ZIP file: https://github.com/wbrown/novelai-research-tool/archive/refs/heads/main.zip
* Or `git clone https://github.com/wbrown/novelai-research-tool.git`

Once you have the source code, in the source directory, do:
  * `go get -u`
  * `go build nrt.go`

This will produce a binary `nrt` file.

Setup
-----
The `nrt` tool uses environment variables to hold your NovelAI username and
password.  They are:
  * `NAI_USERNAME`
  * `NAI_PASSWORD`

**Windows:**

On Windows, type the following in at the command prompt:
```
setx NAI_USERNAME username@email.com
setx NAI_PASSWORD password
```

**MacOS/Linux:**
* On MacOS, edit the `.zshrc` file in your home directory..
* On Linux, eidt the `.profile` file in your home directory.

Add the following lines:
```
export NAI_USERNAME="username@email.com"
export NAI_PASSWORD="password"
```

Either re-login, or restart your terminal, or type the above two lines directly
into your shell prompt.

Running
-------
There is a test file in `tests/need_help.json` that you can run, by invoking:

* Windows: `nrt tests/need_help.json`
* MacOS/Linux:  `./nrt tests/need_help.json`

This will generate multiple output files in `tests` after about 30 minutes,
each containing 10 iterations of 10 generations each.

Details of `test.json`
----------------------
The `nrt` tool accepts a single filename as an argument, the `.json` file
containing test parameters. They are more or less self-explanatory, but I
will highlight some specific ones:
  * `prompt_filename` - where to get the prompts, this is a `txt` file for 
    easy editing of prompts without having to escape like you would in JSON.
  * `output_filename` - where you want the JSON output from the generations to
    go.
  * `iterations` - how many times to run the test, effectively.
  * `generations` - how many times to take the output, concatenate, and re-feed
    back into the AI, like an user.
  * `parameters` - contains NovelAI configuration parameters according to the
    API's specifications.

The sample `tests/need_help.json` can be used as a template, along with a
`tests/need_help.txt` prompt file. There is also an example
`tests/need_help_output_6B-vanilla.json` file that contains an example of what
output `nrt` produces.

Permutation
-----------
There is a special field of the `.json` file, `permutations`. Most of the
fields under `parameters` have a list counterpart under `permutations`.

For example, `model` might look like:
```json
{ "permutations": { "model": [ "2.7B", "6B-v3" ] } }
```
This will cause the `nrt` tool to perform iterations acrosa both models.

If we add another field to permute on, such as `prefix`, it becomes a
combinatorial exercise -- the below `permutations` example will generate
`28` combinations tests to perform:
```json
{ "permutations": {
    "model": [ "2.7B", "6B-v3" ],
    "prefix": [ "vanilla", "theme_naval", "theme_egypt", "theme_dragons",
                "theme_mars", "theme_dragons", "theme_libraries",
                "style_hplovecraft", "style_edgarallanpoe",
                "style_epic_fantasy", "style_slice_of_life",
                "style_romantic", "style_lighthearted_fantasy",
                "style_mmo" ] } }
```

Another example is if we wanted to permute on the `temperature` value:
```json
{ "permutations": {
  "temperature": [ 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8 ]
}}
```

Configuration Notes
-------------------
As of the writing of this section:

The `prefix` takes the following values and requires the `model` attribute to
have the `6B-v3` value:
* `vanilla`
* `theme_naval`
* `theme_egypt`
* `theme_dragons`
* `theme_mars`
* `theme_libraries`
* `style_hplovecraft`
* `style_edgarallanpoe`
* `style_epic_fantasy`
* `style_slice_of_life`
* `style_romantic`
* `style_lighthearted_fantasy`
* `style_mmo`

The `model` parameter takes the following values:
* `2.7B`
* `6B`
* `6B-v3`
