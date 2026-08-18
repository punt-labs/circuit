{
  description = "circuit: tiny state-machine engine for agent workflow loops";

  inputs = {
    nixpkgs.url = "nixpkgs/nixpkgs-unstable";
    beads.url = "github:punt-labs/beads/factory-fixes";
    beads.inputs.nixpkgs.follows = "nixpkgs";
  };

  outputs = { self, nixpkgs, beads }:
    let
      supportedSystems = [
        "aarch64-darwin"
        "x86_64-darwin"
        "aarch64-linux"
        "x86_64-linux"
      ];

      probcliSystems = [
        "aarch64-darwin"
        "x86_64-darwin"
        "x86_64-linux"
      ];

      forAllSystems = nixpkgs.lib.genAttrs supportedSystems;
      forProbcliSystems = nixpkgs.lib.genAttrs probcliSystems;
      probcliSupported = system: builtins.elem system probcliSystems;
    in
    {
      packages = forProbcliSystems (system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
        in
        {
          probcli = pkgs.callPackage ./nix/probcli.nix {};
        });

      devShells = forAllSystems (system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
          bd = beads.packages.${system}.default;
          probcli = if probcliSupported system then self.packages.${system}.probcli else null;
        in
        {
          default = pkgs.mkShell {
            name = "circuit-dev";

            packages = with pkgs; [
              bashInteractive
              bd
              coreutils
              curl
              direnv
              fd
              gh
              git
              gnumake
              go_1_26
              go-tools
              gopls
              jq
              markdownlint-cli2
              nodejs_24
              ripgrep
              shellcheck
            ] ++ pkgs.lib.optionals (probcliSupported system) [ probcli ];

            shellHook = ''
              export CIRCUIT_NIX_SHELL=1
              echo "circuit dev shell: Go $(${pkgs.go_1_26}/bin/go version), bd $(${bd}/bin/bd --version)"
            '';
          };
        });
    };
}
