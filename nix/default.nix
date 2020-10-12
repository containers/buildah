{ system ? builtins.currentSystem }:
let
  pkgs = (import ./nixpkgs.nix {
    config = {
      packageOverrides = pkg: {
        gpgme = (static pkg.gpgme);
        libassuan = (static pkg.libassuan);
        libgpgerror = (static pkg.libgpgerror);
        libseccomp = (static pkg.libseccomp);
        glib = (static pkg.glib).overrideAttrs(x: {
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
    doCheck = false;
    configureFlags = (x.configureFlags or []) ++ [
      "--without-shared"
      "--disable-shared"
    ];
    dontDisableStatic = true;
    enableSharedExecutables = false;
    enableStatic = true;
  });

  self = with pkgs; buildGoModule rec {
    name = "buildah";
    src = ./..;
    vendorSha256 = null;
    doCheck = false;
    enableParallelBuilding = true;
    outputs = [ "out" ];
    nativeBuildInputs = [ bash git go-md2man installShellFiles makeWrapper pkg-config which ];
    buildInputs = [ glibc glibc.static gpgme libassuan libgpgerror libseccomp libapparmor libselinux glib ];
    prePatch = ''
      export CFLAGS='-static'
      export LDFLAGS='-s -w -static-libgcc -static'
      export EXTRA_LDFLAGS='-s -w -linkmode external -extldflags "-static -lm"'
      export BUILDTAGS='static netgo exclude_graphdriver_btrfs exclude_graphdriver_devicemapper seccomp apparmor selinux'
    '';
    buildPhase = ''
      patchShebangs .
      make bin/buildah
    '';
    installPhase = ''
      install -Dm755 bin/buildah $out/bin/buildah
    '';
  };
in self
