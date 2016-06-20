# Contributing guide

## Building

### Requirements

* gvp

### Setup environment

   ```sh
   mkdir -p GrimReaper/src/github.com/matee911 GrimReaper/bin GrimReaper/pkg
   cd GrimReaper/src/github.com/matee911
   git clone https://github.com/matee911/GrimReaper.git
   cd ../../..
   source gvp in
   ```

### Build

   ```sh
   go build
   ```

### Changes.md

Update `CHANGES.md` file.

### Releasing

    ```sh
    $ git tag
    0.1
    0.1.0-alpha2
    0.1.0a1
    $ git tag -a 0.1 -m "message"
    $ git push origin --tags

		# if you forgot about something then...

    $ git tag -d 0.1
    $ git push origin :refs/tags/0.1
    ```
