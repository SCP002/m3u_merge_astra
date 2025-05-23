# m3u_merge_astra

## What is this?

Heavily configurable CLI tool to add channels from M3U playlist into Cesbo Astra 5.

## Why was it made?

To simplify the process of adding new, updating and removing astra streams and it's inputs.

## How it works?

It connects to astra using API, reads its config (mostly streams), adds M3U channels into it from `--m3uPath` by rules
defined in `--programCfgPath` and sends requests to astra to modify config.

## How to use it?

| Command argument     | Description                                                                                     |
| -------------------- | ----------------------------------------------------------------------------------------------- |
| -v, --version        | Print the program version                                                                       |
| -h, --help           | Print help message                                                                              |
| -n, --noninteractive | Do not ask user for input (confirmations etc.)                                                  |
| -l, --logLevel       | Logging level. Can be from `1` (most verbose) to `7` (least verbose) [default: `3`]             |
| -f, --logFile        | Log file. If set, writes structured log to a file at the specified path                         |
| -c, --programCfgPath | Program config file path to read from or initialize a default [default: `m3u_merge_astra.yaml`] |
| -m, --m3uPath        | M3U file path to get channels from. Can be a local file or URL                                  |
| -a, --astraAddr      | Astra address in format of `scheme://host:port` [default: `http://127.0.0.1:8000`]              |
| -u, --astraUser      | Astra user                                                                                      |
| -p, --astraPwd       | Astra password                                                                                  |

Unless `--programCfgPath` is specified, on first run it creates default program config in current directory and terminates.
Tweak it to suit your needs and start the program again.

## Downloads

See [releases page](https://github.com/SCP002/m3u_merge_astra/releases)

## Tips

* `--version` and `--help` goes to **stdout**.  
  Logs goes to **stderr**.

* It is possible to add streams from one instance of astra to another one, for example:  

  ```sh
  m3u_merge_astra -m http://another_astra:8005/playlist.m3u8 -a 127.0.0.1:8002 -u admin -p admin
  ```

* It is possible to use dummy M3U file to run independent tasks
  such as uniting inputs, removing dead ones, filtering etc., for example:

  ```sh
  touch dummy.m3u
  m3u_merge_astra -m dummy.m3u -u admin -p admin
  ```

* When `streams.remove_dead_inputs` is enabled, progress of removing dead inputs from streams is printed every 30 seconds.

## Program config settings

* `general`  
  General settings of the program.

  * `full_translit`  
    Use name transliteration to detect which M3U channel corresponds a stream?

  * `full_translit_map`  
    Source to destination character mapping.  
    All symbols are lowercase as comparsion function will convert every character in a name to lowercase.  
    Key: From. Value: To.

  * `similar_translit`  
    Use name transliteration between visually similar characters to detect which M3U channel corresponds a stream?

  * `similar_translit_map`  
    Source to destination character mapping.
    All symbols are lowercase as comparsion function will convert every character in a name to lowercase.  
    Key: From. Value: To.

  * `name_aliases`  
    Use name aliases list to detect which M3U channel corresponds a stream?

  * `name_alias_list`  
    List of lists.
    Names defined here will be considered identical to any other name in the same nested group.
    During comparsion, names will be simplified (lowercase, no special characters except the `+` sign),
    but not transliterated.

  * `astra_api_resp_timeout`  
    Astra API response timeout.

  * `merge_categories`  
    Should duplicated categories be removed with unique groups combined per category?

* `m3u`  
  M3U related settings of the program.

  * `resp_timeout`  
    M3U playlist URL response timeout in seconds.

  * `chann_name_blacklist`  
    List of regular expressions.  
    If any expression match name of a channel, this channel will be removed from M3U input before merging.

  * `chann_group_blacklist`  
    List of regular expressions.  
    If any expression match group of a channel, this channel will be removed from M3U input before merging.  
    It runs after replacing groups by `chann_group_map` so enter the appropriate values.

  * `chann_url_blacklist`  
    List of regular expressions.  
    If any expression match URL of a channel, this channel will be removed from M3U input before merging.

  * `chann_group_map`  
    Invalid to valid M3U channel group mapping.  
    Key: From. Value: To.

* `streams`  
  Astra streams related settings of the program.

  * `added_prefix`  
    New stream name prefix.  
    Safe to set to ''.
    > Why does it exist?  
    > To distinguish between regular streams and streams added by this program.

  * `add_new`  
    Add new astra streams if streams does not contain M3U channel name?

  * `add_groups_to_new`  
    Add groups to new astra streams?

  * `groups_category_for_new`  
    Category name to use for groups of new astra streams.

  * `add_new_with_known_inputs`  
    Add new astra streams if streams contain M3U channel URL?
    > Why does it exist?  
    > To prevent adding duplicate of existing channel under a different name.

  * `make_new_enabled`  
    Make new streams enabled?

  * `new_type`  
    New stream type, can be one of two types:  
    `spts` - Single-Program Transport Stream. Streaming channels to the end users over IP network.  
    `mpts` - Multi-Program Transport Stream. Preparing multiplexes to DVB modulators.

  * `new_keep_active`  
    Delay before stop stream if no active connections for new streams.

  * `disabled_prefix`  
    Disabled stream name prefix.  
    Safe to set to ''.
    > Why does it exist?  
    > To distinguish between streams manually disabled in dashboard and streams disabled by this program.

  * `remove_without_inputs`  
    Remove streams without inputs?  
    It has priority over `disable_without_inputs`.

  * `disable_without_inputs`  
    Disable streams without inputs?

  * `enable_on_input_update`  
    Enable streams if they got new inputs or inputs were updated (but not removed)?

  * `rename`  
    Rename astra streams as M3U channels if their standartized names are equal?
    > Why does it exist?  
    > To make astra streams named exactly as their counterparts in M3U playlist.
    > For example, astra stream 'tv1000' could be renamed to 'TV 1000' if M3U contains the last one.

  * `add_new_inputs`  
    Add new inputs to astra streams if such found in M3U channels?
    > Why does it exist?  
    > To add new inputs to existing astra streams if M3U channel with matching name found but it's URL is not in stream inputs list.

  * `unite_inputs`  
    Move inputs of astra streams with the same names to the first stream found?
    > Why does it exist?  
    > To help in removal of duplicated streams. Works best with `remove_without_inputs` or `disable_without_inputs`.  
    > For example stream 'SPORTS (HD)' with input A and stream 'Sports HD' with input B will result
    > in 'SPORTS (HD)' with inputs A and B, and 'Sports HD' with no inputs.

  * `hash_check_on_add_new_inputs`  
    Add new inputs to astra streams even if M3U channel and stream input only differ by hash (everything after #)?
    > Why does it exist?  
    > To prevent adding already exising URL's to inputs astra astreams.  
    > For example if it set to 'true', M3U channel URL '<http://channel>' will be added to stream with input
    > '<http://channel#no_sync>', otherwise it won't.

  * `sort_inputs`  
    Sort inputs of astra streams?
    > Why does it exist?  
    > To structurize logs and produce astra config which is easier to navigate in.

  * `input_weight_to_type_map`  
    Mapping of how high stream input should appear in the list after sorting.  
    Any unspecified input will have weight of `unknown_input_weight`.

  * `unknown_input_weight`  
    Default weight of unknown inputs.

  * `input_blacklist`  
    List of regular expressions.  
    If any expression match URL of a stream's input, this input will be removed from astra streams before adding new ones.

  * `remove_duplicated_inputs`  
    Remove inputs of astra streams which already present in config?

  * `remove_duplicated_inputs_by_rx_list`  
    List of regular expressions.  
    If any first capture group (anything surrounded by the first `()`) of regular expression match URL of input of a
    stream, any other inputs of that stream which first capture group is the same will be removed from stream.  
    This setting is not controlled by `remove_duplicated_inputs`.
    > Why does it exist?  
    > To be able to remove dulticated inputs per stream by specific conditions, for example by host name.

  * `remove_disabled_inputs`  
    Remove disabled inputs?

  * `disable_all_but_one_input_by_rx_list`  
    List of regular expressions.  
    If any expression match URL of a stream's input, only this input will be kept and all other will be disabled.  
    If list has multiple entries, only input matching first found expression will be kept.
    > Why does it exist?  
    > To give an option to leave only one input in stream, if specific inputs implies so.
    >
    > For example, input with `#no_sync` will periodically be unavailable, causing stream to switch to another input,
    > causing unwanted interruptions.

  * `remove_dead_inputs`  
    Remove inputs of astra streams which do not respond or give invalid response?  
    Supports HTTP(S), enable `use_analyzer` option for more.  
    It has priority over `disable_dead_inputs`.

  * `disable_dead_inputs`  
    Disable inputs of astra streams which do not respond or give invalid response?  
    Supports HTTP(S), enable `use_analyzer` option for more.

  * `dead_inputs_check_blacklist`  
    List of regular expressions.  
    If any expression match URL of a stream's input, this input will not be checked for availability.

  * `input_max_conns`  
    Maximum amount of simultaneous connections to validate inputs of astra streams.  
    Use more than 1 with caution. It may result in false positives if server consider frequent requests as spam.

  * `input_resp_timeout`  
    Astra stream input response timeout.

  * `use_analyzer`  
    Use astra analyzer (astra --analyze -p \<port\>) to check for dead inputs?  
    Supports HTTP(S), UDP, RTP, RTSP.
    > Why does it exist?  
    > To remove or disable dead inputs by advanced criteria such as bitrate or errors, check more protocols and
    > use analyzer as a proxy if needed.

  * `analyzer_addr`  
    Astra analyzer address in format of 'host:port'.

  * `analyzer_watch_time`  
    Amount of time per attempt that astra analyzer should spend collecting results.

  * `analyzer_max_attempts`  
    Maximum amount of attempts that astra analyzer should perform trying to get response from stream input.
    > Why does it exist?  
    > On rare occasions, some streams may not respond after first request no matter how much time analyzer will
    > spend trying to get info from it.

  * `analyzer_bitrate_threshold`  
    Average bitrate threshold in kbit/s for stream inputs.  
    If astra analyzer will return bitrate lower than specified threshold, input will be cosidered dead.

  * `analyzer_video_only_bitrate_threshold`  
    Average bitrate threshold in kbit/s for stream inputs without audio.  
    If astra analyzer will return bitrate lower than specified threshold, input will be cosidered dead.

  * `analyzer_audio_only_bitrate_threshold`  
    Average bitrate threshold in kbit/s for stream inputs without video.  
    If astra analyzer will return bitrate lower than specified threshold, input will be cosidered dead.

  * `analyzer_cc_errors_threshold`  
    Average amount of CC errors for stream inputs.  
    If astra analyzer will return amount of CC errors higher than specified threshold, input will be cosidered dead.  
    Set to negative value to disable this check.

  * `analyzer_pcr_errors_threshold`  
    Average amount of PCR errors for stream inputs.  
    If astra analyzer will return amount of PCR errors higher than specified threshold, input will be cosidered dead.  
    Set to negative value to disable this check.

  * `analyzer_pes_errors_threshold`  
    Average amount of PES errors for stream inputs.  
    If astra analyzer will return amount of PES errors higher than specified threshold, input will be cosidered dead.  
    Set to negative value to disable this check.

  * `input_update_map`  
    List of regular expression pairs.  
    If any `from` expression match URL of astra stream's input, it will be replaced with URL from according M3U
    channel if it matches the `to` expression.  
    Only first matching input will be updated per M3U channel.  
    In most cases specified `from` and `to` should be identical.
    > Why does it exist?  
    > To update (remove old, add new) inputs of astra streams if they were changed in M3U file, but
    > preserve input's order, existing hashes (everything after #) and to produce readable report
    > about which inputs were replaced.  
    >
    > Why not to use `input_blacklist` to remove, `add_new_inputs` to add, **\*_to_input_hash_map**
    > to generate hashes and `sort_inputs` to sort them instead?  
    > Since old inputs will be removed beforehand, original hash will be lost and **\*_to_input_hash_map**
    > might not cover your usecase, the same with bulk sorting.  
    > Also update will be divided into 3 different parts, making it difficult to see actual changes.

  * `update_inputs`  
    Update inputs of astra streams with M3U channels according to `input_update_map`?

  * `keep_input_hash`  
    Keep old input hash on updating inputs of astra streams?

  * `remove_inputs_by_update_map`  
    Remove inputs of astra streams which match at least one `input_update_map.from` expression but not in M3U channels?
    > Why does it exist?  
    > To remove inputs of astra streams which are no longer valid (meant to be updated but not exist anymore).

  * `name_to_input_hash_map`  
    Mapping of stream name regular expression to stream input hash which should be added.
    > Why does it exist?  
    > To be able to add specific hashes per stream name, for example '#buffer_time=...' to HD streams.

  * `group_to_input_hash_map`  
    Mapping of stream group regular expression to stream input hash which should be added.  
    Stream groups should be defined to match expressions in the form of 'Category: Group'.
    > Why does it exist?  
    > To be able to add specific hashes per stream group, for example '#no_sync' to radio streams.

  * `input_to_input_hash_map`  
    Mapping of stream input regular expression to stream input hash which should be added.
    > Why does it exist?  
    > To be able to add specific hashes per stream input, for example set User-Agent (#ua=...) to specific URL's.

  * `name_to_keep_active_map`  
    Mapping of stream name regular expression to `keep active` setting of stream which should be set.  
    Only first matching rule applies per stream in the priority: By inputs -> By name -> By group.
    > Why does it exist?  
    > To be able to add `keep active` setting per stream name, for example 5 to frequently requested streams or 0 to disable it.

  * `group_to_keep_active_map`  
    Mapping of stream group regular expression to `keep active` setting of stream which should be set.  
    Only first matching rule applies per stream in the priority: By inputs -> By name -> By group.
    > Why does it exist?  
    > To be able to add `keep active` setting per stream group, for example 5 to frequently requested streams or 0 to disable it.

  * `input_to_keep_active_map`  
    Mapping of stream input regular expression to `keep active` setting of stream which should be set.  
    Only first matching rule applies per stream in the priority: By inputs -> By name -> By group.  
    Setting will be set if at least one input matches the `by` expression.
    > Why does it exist?  
    > To be able to add `keep active` setting per stream input, for example 5 to frequently requested streams or 0 to disable it.

## Build from source code [Go / Golang]

1. Install [Golang](https://golang.org/) 1.23 or newer.

2. Download the source code:  

    ```sh
    git clone https://github.com/SCP002/m3u_merge_astra.git
    ```

3. Install dependencies:

    ```sh
    cd src
    go mod tidy
    ```

    Or

    ```sh
    cd src
    go get ./...
    ```

4. Update dependencies (optional):

    ```sh
    go get -u ./...
    ```

5. To build a binary for current OS / architecture into `../build/` folder:

    ```sh
    go build -o ../build/ m3u_merge_astra.go
    ```

    Or run `/src/build.sh` to build binaries for every OS / architecture pair.
