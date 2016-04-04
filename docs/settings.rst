Transport configuration:
------------------------

Every connection in gofast is encapsulated as a Transport{} object
returned by NewTransport() API.

Following is the list of settings that can be applied on a single
transport.

`name`
    give a name for the transport.

`buffersize`
    maximum size that a single message will need.

`batchsize`
    number of messages to batch before writing to socket, transport
    will create a local buffer of size buffersize * batchsize.

`chansize`
    channel size to use for internal go-routines.

`opaque.start`
    starting opaque range, inclusive.

`opaque.end`
    ending opaque range, inclusive.

`tags`
    comma separated list of tags to apply, in specified order.

`log.level`
    log level to use for default-logger.

`log.file`
    log file to use for default-logger, if empty stdout is used.

`gzip.level`
    gzip compression level, if `tags` contain "gzip".
