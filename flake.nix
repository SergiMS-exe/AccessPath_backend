{
  description = "AccessPath backend: Go 1.25+ / PostgreSQL / Redis / DuckDB";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
  };

  outputs = { self, nixpkgs }:
    let
      system = "x86_64-linux";
      pkgs = import nixpkgs { inherit system; };

      tools = with pkgs; [
        go
        gopls
        gotools # goimports
        govulncheck
        golangci-lint
        air
        git
        curl
        jq
        unzip
        gnumake
        postgresql_16
        redis
        duckdb
        docker-client
      ];
    in
    {
      devShells.${system} = {
        default = pkgs.mkShell {
          name = "accesspath-backend";
          packages = tools;
          shellHook = ''
            export GOFLAGS="-mod=mod"

            # El repo usa .golangci.yml en formato v1; el binario de nixpkgs es
            # v2 y no lo lee. Este helper fija la version que usa CI.
            golangci-lint-v1() {
              go run github.com/golangci/golangci-lint/cmd/golangci-lint@v1.62.2 run "$@"
            }

            # Genera docs/ con la misma version que CI.
            swag-init() {
              go run github.com/swaggo/swag/cmd/swag@v1.16.3 init -g cmd/server/main.go
            }

            echo "accesspath-backend: Go $(go version | cut -d' ' -f3)"
            echo "  make migrate-up | air | go test ./... -race"
            echo "  ETL: DATABASE_URL=... ./scripts/import_overture.sh"
          '';
        };
      };

      formatter.${system} = pkgs.nixfmt;
    };
}
