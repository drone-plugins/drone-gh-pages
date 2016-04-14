Use the gh-pages plugin to build and push a directory from your build to your repo's GitHub Pages.
The following parameters are used to configure this plugin:

* `upstream` - name of the upstream to push changes to (default origin)
* `pagesDirectory` - the directory of your build that should become your pages content (default docs)
* `temporaryBaseDirectory` - the temporary directory used to store the gh-pages clone (default .tmp)
* `target` - the branch to publish to (default gh-pages)

The following is a sample configuration in your .drone.yml file:

```yaml
publish:
  gh-pages:
    pagesDirectory: pages
    upstream: origin
```

## Known Issues

At this time, this plugin will not allow you to create the gh-pages branch if it does not exist.
