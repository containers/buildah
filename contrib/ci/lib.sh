OS_RELEASE_VER="$(source /etc/os-release; echo $VERSION_ID | tr -d '.')"
OS_RELEASE_ID="$(source /etc/os-release; echo $ID)"
OS_REL_VER="$OS_RELEASE_ID-$OS_RELEASE_VER"


function die() {
    echo "$1" >&2
    exit 1
}

function parse_args() {
    TEST=
    DISTRO_NAME=
    STORAGE_DRIVER=overlay
    PRIV=root
    case "$#" in
        2)
            TEST=$1
            DISTRO_NAME=$2
            ;;
        3)
            TEST=$1
            PRIV=$2
            DISTRO_NAME=$3
            ;;
        4)
            TEST=$1
            STORAGE_DRIVER=$2
            PRIV=$3
            DISTRO_NAME=$4
            ;;
        *)
            die "Invalid number of arguments $#, need 2-4"
            ;;
    esac

    validate_distro "$DISTRO_NAME"
    validate_storage "$STORAGE_DRIVER"
    validate_priv "$PRIV"
}

function validate_distro() {
    case "$1" in
        "fedora-current"|"fedora-prior"|"fedora-rawhide"|"debian-sid")
            ;;
        *)
            die "Unknown DISTRO_NAME '$1' set"
            ;;
    esac
}

function validate_storage() {
    case "$1" in
        "vfs"|"overlay")
            ;;
        *)
            die "Unknown STORAGE_DRIVER '$1' set"
            ;;
    esac
}

function validate_priv() {
    case "$1" in
        "root"|"rootless")
            ;;
        *)
            die "Unknown PRIV '$1' set"
            ;;
    esac
}
