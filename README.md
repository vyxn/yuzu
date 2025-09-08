# TODO

## Server

### Provider
- [x] migrate comicvine to generic provider
- [x] migrate kitsu to generic provider
- [x] optimize provider setup
- [x] check provider config autoreloading
- [x] .config/meta-yuzu/providers/*.json discover all jsons in there
  - [x] on setup create folder structure + copy the ones in project for the time being
- [x] xml response support
- [x] include json path library to access results
- [x] return compiled output value
- [x] handle comicinfo schema validation
- [ ] add xpath for xml only APIs
- [ ] improve generic provider Run()
- [x] validate expected provider json with a schema, created from our structs
- [x] add support for yaml output
- [x] add support for md frontmatter output
- [x] handle querying all providers
- [x] allow to delete a provider
- [ ] rate limiting for provider

### Library
- [x] make basic structure of a library
- [x] define library jobs structure
- [x] see how to link with providers
- [x] allow scheduling of jobs with crontab syntax
- [x] handle querying all libraries and a single library
- [ ] allow create and update a library
- [ ] allow delete a library
- [ ] add run job now
- [ ] optimize number of provider runs with gorutines

### Additional Providers
- [ ] books: goodreads
- [ ] books: hardcover

