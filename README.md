# Nearwait

This is Nearwait, a tool for managing your project files.

## What does it do?

Creates txtar with all files you enabled in .manifest.yml

So, first create manifest in your new project:

```bash
cd myproject
nearwait manifest
```

Now we have .manifest.yml.  Comment out the files you don't want listed in txtar file.

Run nearwait again to generate txtar


## Quick Start

Here's how to get started with Nearwait:

```bash
# Generate a manifest
./nearwait manifest

# Force overwrite an existing manifest
./nearwait manifest --force

# Use a custom manifest file name
./nearwait manifest --manifest my_manifest.yml

# Run in debug mode (keeps temp files)
./nearwait manifest --debug

# Combine flags
./nearwait manifest --force --debug --manifest custom_manifest.yml
```

## Cheat Sheet

| Command | Description |
|---------|-------------|
| `./nearwait manifest` | Generate a manifest file |
| `./nearwait manifest --force` | Overwrite existing manifest |
| `./nearwait manifest --manifest FILE` | Use custom manifest file name |
| `./nearwait manifest --debug` | Run in debug mode |

## Examples

1. Basic usage:
   ```
   ./nearwait manifest
   ```

2. Force regenerate manifest:
   ```
   ./nearwait manifest --force
   ```

3. Use a custom manifest name and debug:
   ```
   ./nearwait manifest --manifest project_files.yml --debug
   ```
