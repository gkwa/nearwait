# Nearwait

Nearwait copies project files to the clipboard according to what's specified in a local manifest YAML file.

## Motivation

Working with GPT to iterate on editing project files.

## Workflow

1. Recursively scans the current directory for files
1. Generates a manifest file (default: `.nearwait.yml`) with all files initially commented out
1. Allows user to manually edit the manifest to enable specific files
1. Generates a txtar archive (default `.nearwait.txtar`) based on the enabled files in the manifest
1. Automatically copies the txtar content to the clipboard when enabled files are present

## Usage

1. Run Nearwait in your project directory:
   ```
   nearwait
   ```
   This generates the initial `.nearwait.yml` manifest with all files commented out.

1. Edit the `.nearwait.yml` file to uncomment (enable) the files you want to include:
   ```yaml
   filelist:
   # - /path/to/excluded/file.txt
   - /path/to/included/file.txt
   ```

1. Run Nearwait again to process the manifest and generate the txtar archive:
   ```
   nearwait
   ```

1. If there are enabled files in the manifest, the txtar content will be automatically copied to your clipboard.

## Options

- `--force`: Force overwrite of existing manifest
- `--debug`: Keep temporary directory for debugging
- `--manifest <filename>`: Specify a custom name for the manifest file (default: `.nearwait.yml`)
- `--verbose`, `-v`: Enable verbose mode
- `--log-format`: Set log format to 'json' or 'text' (default is text)
- `--config`: Specify a config file (default is $HOME/.nearwait.yaml)

## Notes

- The tool ignores certain directories by default (e.g., `.git`, `node_modules`, etc.)
- The txtar archive is named based on the manifest filename (e.g., `.nearwait.txtar` for the default manifest)

## Installation

To install Nearwait, ensure you have Go installed on your system, then run:

```
go install github.com/gkwa/nearwait@latest
```

## Building from Source

To build Nearwait from source:

1. Clone the repository:
   ```
   git clone https://github.com/gkwa/nearwait.git
   ```

1. Navigate to the project directory:
   ```
   cd nearwait
   ```

1. Build the project:
   ```
   go build
   ```
