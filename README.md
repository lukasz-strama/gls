# gls

A modern alternative to the GNU `ls` utility, written in Go. Propably usless but fun to make.

## Features
- Colorized output based on file types
- Optional icons for files and directories
- Sorting options (by name, size, modification time, extension)
- Size filtering (minimum and maximum size)
- Configuration file support for default options

### Flags

- `-l`: Use a long listing format
- `-a`: Do not ignore entries starting with .
- `-icons`: Show icons
- `-sort`: Sort by field: name, size, time, ext (default "name")
- `-min-size`: Filter by minimum size (e.g. 10K, 1M)
- `-max-size`: Filter by maximum size (e.g. 10K, 1M)
- `-dirs-first`: List directories first

## Configuration

`gls` supports a configuration file located at `~/.config/gls/config.json`. You can use this file to set default options.

Example `config.json`:

```json
{
  "icons": true,
  "dirs_first": true,
  "long": false,
  "all": false,
  "sort": "name",
  "min_size": "",
  "max_size": ""
}
```
