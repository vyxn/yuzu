env "local" {
  url = "sqlite://meta.db"
  migration {
    dir = "file://migrations"
  }
}
