{ self, ... }:
{
  perSystem = { pkgs, ... }:
  let
    rmmock = pkgs.callPackage ./rmmock {
      src = pkgs.nix-gitignore.gitignoreSource [] self;
    };
    default = rmmock;
  in {
    packages = {
      inherit rmmock default;
    };
  };
}