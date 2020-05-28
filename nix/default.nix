{ system ? builtins.currentSystem }:
let
  pkgs = (import ./nixpkgs.nix {
    config = {
      packageOverrides = pkg: {
        gpgme = (static pkg.gpgme);
        libassuan = (static pkg.libassuan);
        libgpgerror = (static pkg.libgpgerror);
        libseccomp = (static pkg.libseccomp);
        glib = pkg.glib.overrideAttrs(x: {
          outputs = [ "bin" "out" "dev" ];
          mesonFlags = [
            "-Ddefault_library=static"
            "-Ddevbindir=${placeholder ''dev''}/bin"
            "-Dgtk_doc=false"
            "-Dnls=disabled"
          ];
        });
      };
    };
  });

  static = pkg: pkg.overrideAttrs(x: {
    configureFlags = (x.configureFlags or []) ++
      [ "--without-shared" "--disable-shared" ];
    dontDisableStatic = true;
    enableSharedExecutables = false;
    enableStatic = true;
  });

  self = with pkgs; buildGoPackage rec {
    name = "buildah";
    src = ./..;
    goPackagePath = "github.com/containers/buildah";
    doCheck = false;
    enableParallelBuilding = true;
    nativeBuildInputs = [ git installShellFiles pkg-config ];
    buildInputs = [ glib glibc glibc.static gpgme libapparmor libassuan libgpgerror libseccomp libselinux ];
    prePatch = ''
      export LDFLAGS='-s -w -static-libgcc -static'
      export EXTRA_LDFLAGS='-s -w -linkmode external -extldflags "-static -lm"'
      export BUILDTAGS='static netgo apparmor selinux seccomp exclude_graphdriver_btrfs exclude_graphdriver_devicemapper'
    '';
    buildPhase = ''
      pushd go/src/${goPackagePath}
      patchShebangs .
      make bin/buildah
    '';
    installPhase = ''
      install -Dm755 bin/buildah $out/bin/buildah
    '';
  };
in self
