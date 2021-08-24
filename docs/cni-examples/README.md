When [buildah](https://github.com/containers/buildah)'s `buildah run`
command is used, or when  `buildah build` needs to handle a
`RUN` instruction, the processes which `buildah` starts are run in their own
network namespace unless the `--network=host` option is used.

When a network namespace is first created, it contains no network interfaces
and is essentially disconnected from any networks that the host can access.

In order to configure network interfaces and network access for those network
namespaces, `buildah` uses the
[CNI](https://github.com/containernetworking/cni) library, which in turn uses
plugins ([CNI plugins](https://github.com/containernetworking/plugins), and
possibly others).

Which plugins get used, and how, is controlled using configuration files, which
`buildah` scans `/etc/cni/net.d` to find.  By default, `buildah` expects to
find plugins in `/opt/cni/bin`.

This directory contains sample configuration files for the `loopback` and
`bridge` plugins from the [CNI
plugins](https://github.com/containernetworking/plugins) repository.  To
install those plugins, try running:

```
  git clone https://github.com/containernetworking/plugins
  ( cd ./plugins; ./build.sh )
  mkdir -p /opt/cni/bin
  install -v ./plugins/bin/* /opt/cni/bin
```

If you've already installed a CNI configuration (for example, for
[CRI-O](https://github.com/kubernetes-sigs/cri-o)), it'll probably just
work, but to install these sample configuration files:
```
  mkdir -p /etc/cni/net.d
  install -v -m644 *.conf /etc/cni/net.d/
```
