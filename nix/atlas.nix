{ pkgs }:
# Atlas CLI (Official Binary)
# We use the official binary from ariga.io instead of the community version
# because the community version (nixpkgs) does not support the 'ent://' schema loader.
let
  pname = "atlas";
  version = "0.38.0";

  # Define hashes for supported systems
  hashes = {
    x86_64-linux = "sha256-JuyFy6NiOV9PXznbB/ggQmi0w54+R4pm8+d8wJjDC6s=";
    aarch64-linux = "sha256-ZKiUcksYBueeRqUUPGTag6kKrbh/CB8E3ZAXjrk4DSw=";
    x86_64-darwin = "sha256-3ueyD5k0ArjCnjKUqIH4WedTAba9qfoQjkeja3nfZIE=";
    aarch64-darwin = "sha256-m7F4pl1IsVY1LONkGKCRy4eka3PEg9b9wz4PJ9BnwhQ=";
  };

  # Map nixpkgs systems to atlas platform names
  systems = {
    x86_64-linux = "linux-amd64";
    aarch64-linux = "linux-arm64";
    x86_64-darwin = "darwin-amd64";
    aarch64-darwin = "darwin-arm64";
  };

  system = pkgs.stdenv.hostPlatform.system;
  platform = systems.${system} or (throw "Unsupported system: ${system}");
  hash = hashes.${system} or (throw "Unsupported system: ${system}");

in
pkgs.stdenv.mkDerivation rec {
  inherit pname version;

  src = pkgs.fetchurl {
    url = "https://release.ariga.io/atlas/atlas-${platform}-v${version}";
    inherit hash;
  };

  dontUnpack = true;

  installPhase = ''
    runHook preInstall
    install -D $src $out/bin/atlas
    chmod +x $out/bin/atlas
    runHook postInstall
  '';
}
