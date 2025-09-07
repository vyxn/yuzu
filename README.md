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
- [ ] validate expected provider json with a schema

### Library
- [ ] make basic structure of a library
- [ ] define library jobs structure
- [ ] see how to link with providers
- [ ] allow scheduling of jobs with crontab syntax
