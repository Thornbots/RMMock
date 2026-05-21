{
  src,
  lib,
  buildGoModule,
  pkg-config,
  opencv,
  ffmpeg,
}:
let
  pname = "rmmock";
  custom-opencv = opencv.override {
    enableGtk3 = true;
  };
in buildGoModule {
  version = "0.1.0";
  meta = with lib; {
    description = "RMMock - A mock server for RoboMaster competition";
    homepage = "https://github.com/stydxm/RMMock";
    license = licenses.mit;
    maintainers = [ "stydxm" "vix_hentx" ];
  };

  inherit pname src;

  vendorHash = "sha256-kNQQ6+sqdnPxSeNVWgD1o+pdVquRSVHeyaY1g/KVoac=";

  nativeBuildInputs = [ pkg-config ];
  buildInputs = [ custom-opencv ffmpeg ];

  preBuild = ''
    export PKG_CONFIG_PATH="${custom-opencv}/lib/pkgconfig:${ffmpeg.dev}/lib/pkgconfig:$PKG_CONFIG_PATH"
  '';

  postInstall = ''
    ln -s $out/bin/RMMock $out/bin/${pname}
  '';

}
