This subdirectory contains a script used to create images for testing.

To rephrase: this script is used **before testing**, not used **in** testing.
_Much_ before testing (days/weeks/months/years), and manually.

The script is `make-v2sN` but it is never invoked as such. Instead,
various different symlinks point to the script, and the script
figures out its use by picking apart the name under which it is called.

As of the initial commit on 2020-02-10 there are three symlinks:

* make-v2s1 - Create a schema 1 image
* make-v2s2 - Create a schema 2 image
* make-v2s1-with-dups - Create a schema 1 image with two identical layers

If the script is successful, it will emit instructions on how to
push the images to quay and what else you might need to do.

Updating
========

Should you need new image types, e.g. schema version 3 or an image
with purple elephant GIFs in it:

1. Decide on a name. Create a new symlink pointing to `make-v2sN`
1. Add the relevant code to `make-v2sN`: a conditional check at the top, the actual image-creating code, and if possible a new test to make sure the generated image is good
1. Run the script. Verify that the generated image is what you expect.
1. Add new test(s) to `digest.bats`
