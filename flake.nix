{
  inputs = {
    nixpkgs.url = "nixpkgs/nixos-unstable";
    utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, utils }:
    utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs { inherit system; };
        version = builtins.substring 0 8 self.lastModifiedDate;
      in
      {
        defaultPackage = pkgs.buildGoModule {
          pname = "shotor";
          inherit version;

          src = ./.;

          CGO_ENABLED = "0";

          # vendorSha256 = pkgs.lib.fakeSha256;
          vendorSha256 = "sha256-pQpattmS9VmO3ZIQUFn66az8GSmB4IvYhTTCFn6SUmo=";
        };

        defaultApp = utils.lib.mkApp { drv = self.defaultPackage."${system}"; };

        devShell = with pkgs; mkShell { buildInputs = [ go_1_18 ]; };
      }
    );
}
