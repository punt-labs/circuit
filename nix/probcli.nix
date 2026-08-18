# probcli derivation for circuit's dev shell and CI.
#
# ProB is not in nixpkgs. This derivation fetches the official release
# archives from stups.hhu-hosting.de (linked from prob.hhu.de) and extracts the
# probcli binary.
#
# Platforms:
#   Linux x86_64:  ProB.linux64.tar.gz
#   macOS (universal arm/intel): ProB.macos.zip
#
# ProB does not publish Linux aarch64 binaries. CI must use x86_64.
#
# To update: change version and update both sha256 hashes using
#   nix-prefetch-url --type sha256 <url>
{ lib, stdenv, fetchurl, unzip, autoPatchelfHook, zlib, glibc, gcc-unwrapped, util-linux, gmp }:

let
  version = "1.15.1";
  base = "https://stups.hhu-hosting.de/downloads/prob/tcltk/releases/${version}";

  linux = fetchurl {
    url = "${base}/ProB.linux64.tar.gz";
    sha256 = "092c6ins8p5lflacy49j4b2crp6p1zpz766qqbyykkw13kvi3m5h";
  };

  macos = fetchurl {
    url = "${base}/ProB.macos.zip";
    sha256 = "1hi4izsgz6n0z3g44708z7sgnjywn69pcwxiyhc7zv2nxs7ci1bf";
  };
in

assert stdenv.isDarwin || stdenv.hostPlatform.system == "x86_64-linux";

stdenv.mkDerivation {
  pname = "probcli";
  inherit version;

  src = if stdenv.isDarwin then macos else linux;

  nativeBuildInputs = lib.optionals stdenv.isLinux [
    autoPatchelfHook
    unzip
  ] ++ lib.optionals stdenv.isDarwin [
    unzip
  ];

  buildInputs = lib.optionals stdenv.isLinux [
    zlib
    glibc
    gcc-unwrapped.lib
    util-linux.lib
    gmp
  ];

  unpackPhase = if stdenv.isDarwin
    then "unzip $src"
    else "tar xzf $src";

  installPhase = ''
    # ProB needs lib/, stdlib/, and tcl/ alongside the binary.
    # Install everything into $out/prob and add a wrapper in $out/bin.
    mkdir -p $out/prob $out/bin
    if [ -d ProB ]; then
      cp -r ProB/. $out/prob/
    else
      cp -r . $out/prob/
    fi
    chmod +x $out/prob/probcli
    # Wrapper so probcli resolves its lib relative to the installation.
    cat > $out/bin/probcli <<EOF
#!${stdenv.shell}
exec $out/prob/probcli "\$@"
EOF
    chmod +x $out/bin/probcli
  '';

  dontFixup = stdenv.isDarwin;
  dontStrip = true;

  meta = {
    description = "ProB command-line model checker for B and Event-B";
    homepage = "https://prob.hhu.de";
    platforms = [ "x86_64-linux" "x86_64-darwin" "aarch64-darwin" ];
    # ProB is proprietary. Binary redistribution not permitted.
    # This derivation fetches from the official upstream URL only.
  };
}
