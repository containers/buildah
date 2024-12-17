# buildah-mkcw "1" "July 2023" "buildah"

## NAME
buildah\-mkcw - Convert a conventional container image into a confidential workload image.

## SYNOPSIS
**buildah mkcw** [*options*] *source* *destination*

## DESCRIPTION
Converts the contents of a container image into a new container image which is
suitable for use in a trusted execution environment (TEE), typically run using
krun (i.e., crun built with the libkrun feature enabled and invoked as *krun*).
Instead of the conventional contents, the root filesystem of the created image
will contain an encrypted disk image and configuration information for krun.

## source
A container image, stored locally or in a registry

## destination
A container image, stored locally or in a registry

## OPTIONS

**--add-file** *source[:destination]*

Read the contents of the file `source` and add it to the committed image as a
file at `destination`.  If `destination` is not specified, the path of `source`
will be used.  The new file will be owned by UID 0, GID 0, have 0644
permissions, and be given a current timestamp.  This option can be specified
multiple times.

**--attestation-url**, **-u** *url*
The location of a key broker / attestation server.
If a value is specified, the new image's workload ID, along with the passphrase
used to encrypt the disk image, will be registered with the server, and the
server's location will be stored in the container image.
At run-time, krun is expected to contact the server to retrieve the passphrase
using the workload ID, which is also stored in the container image.
If no value is specified, a *passphrase* value *must* be specified.

**--base-image**, **-b** *image*
An alternate image to use as the base for the output image.  By default,
the *scratch* non-image is used.

**--cpus**, **-c** *number*
The number of virtual CPUs which the image expects to be run with at run-time.
If not specified, a default value will be supplied.

**--firmware-library**, **-f** *file*
The location of the libkrunfw-sev shared library.  If not specified, `buildah`
checks for its presence in a number of hard-coded locations.

**--memory**, **-m** *number*
The amount of memory which the image expects to be run with at run-time, as a
number of megabytes.  If not specified, a default value will be supplied.

**--passphrase**, **-p** *text*
The passphrase to use to encrypt the disk image which will be included in the
container image.
If no value is specified, but an *--attestation-url* value is specified, a
randomly-generated passphrase will be used.
The authors recommend setting an *--attestation-url* but not a *--passphrase*.

**--slop**, **-s** *{percentage%|sizeKB|sizeMB|sizeGB}*
Extra space to allocate for the disk image compared to the size of the
container image's contents, expressed either as a percentage (..%) or a size
value (bytes, or larger units if suffixes like KB or MB are present), or a sum
of two or more such specifications.  If not specified, `buildah` guesses that
25% more space than the contents will be enough, but this option is provided in
case its guess is wrong.  If the specified or computed size is less than 10
megabytes, it will be increased to 10 megabytes.

**--type**, **-t** {SEV|SNP}
The type of trusted execution environment (TEE) which the image should be
marked for use with.  Accepted values are "SEV" (AMD Secure Encrypted
Virtualization - Encrypted State) and "SNP" (AMD Secure Encrypted
Virtualization - Secure Nested Paging).  If not specified, defaults to "SNP".

**--workload-id**, **-w** *id*
A workload identifier which will be recorded in the container image, to be used
at run-time for retrieving the passphrase which was used to encrypt the disk
image.  If not specified, a semi-random value will be derived from the base
image's image ID.

## SEE ALSO
buildah(1)
