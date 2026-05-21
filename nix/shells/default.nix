{ ... }:
{
  perSystem = { self', pkgs, ... }:
  {
    devShells.default = pkgs.mkShell {
      inputsFrom = [ self'.packages.default ];
      
      nativeBuildInputs = with pkgs; [
        go
        gopls
        gotools
        pkg-config
      ];

      env = let
        GOPATH = "$PWD/.go";
        PATH = "${GOPATH}/bin:$PATH";
      in {
        inherit GOPATH PATH;
      };
    };
  };
}