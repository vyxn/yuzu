# Tools

- https://www.jsonschemavalidator.net
- https://regex101.com
- https://crontab.guru

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
- [x] improve generic provider Run()
- [x] validate expected provider json with a schema, created from our structs
- [x] add support for yaml output
- [x] add support for md frontmatter output
- [x] handle querying all providers
- [x] allow to delete a provider
- [x] rate limiting for provider
- [ ] try if unmarshaling directly from the json schema library works
- [x] implement caching mechanism for repeated calls

### Library
- [x] make basic structure of a library
- [x] define library jobs structure
- [x] see how to link with providers
- [x] allow scheduling of jobs with crontab syntax
- [x] handle querying all libraries and a single library
- [x] allow create and update a library
- [x] allow delete a library
- [x] add run job now

### Job
- [x] make base flow of a job run
- [x] optimize number of provider runs with gorutines
- [ ] job output: to disk file
- [ ] job output: cbz inclusion
- [ ] job output: merge multiple provider results

### Additional Providers
- [ ] books: goodreads
- [ ] books: hardcover

