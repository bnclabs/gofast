Following list of CBOR_ tags are reserved for gofast protocol. Refer to the
`README <../README.rst>`_ page for more information.

`tag-6`
    tagPost, following tagged CBOR byte-array carries a POST request.

`tag-7`
    tagStream, following tagged CBOR byte-array carries a STREAM message.

`tag-8`
    tagFinish, following is tagged CBOR breakstop (0xff) item.

`tag-37`
    tagMsg, following CBOR map carries message header and data.

`tag-38`
    tagId, used as key in CBOR header-data mapping to unique message ID.

`tag-39`
    tagData, used as key in CBOR header-data mapping to message, binary
    encoded as CBOR byte-array.

`tag-40`
    tagGzip, following CBOR byte array is compressed using gzip encoding.

`tag-41`
    tagLzw, following CBOR byte array is compressed using gzip encoding.

These reserved tags are not part of CBOR specification or IANA registry,
please refer/follow issue `#1 <https://github.com/prataprc/gofast/issues/1>`_.

.. _CBOR: http://cbor.io/
