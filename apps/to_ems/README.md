# Bhojpur EMS - Stdin to EMS

A tool for publishing to a Bhojpur EMS topic with data from `stdin`.

## Usage

```
Usage of ./to_ems:
  -delimiter string
    	character to split input from stdin (default "\n")
  -emsd-tcp-address value
    	destination EMSd TCP address (may be given multiple times)
  -producer-opt value
    	option to passthrough to ems.Producer (may be given multiple times
  -rate int
    	Throttle messages to n/second. 0 to disable
  -topic string
    	EMS topic to publish to
```
    
### Examples

Publish each line of a file:

```bash
$ cat source.txt | to_ems -topic="topic" -emsd-tcp-address="127.0.0.1:4150"
```

Publish three messages, in one go:

```bash
$ echo "one,two,three" | to_ems -delimiter="," -topic="topic" -emsd-tcp-address="127.0.0.1:4150"
```