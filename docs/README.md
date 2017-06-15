# Object Drive Docs

## Install aglio

**aglio** is a "global" node utility. Install it with

```
npm install -g aglio
```

## Run the aglio server

To start the aglio server for .apib processing, issue this command from this
project's root folder:

```
aglio --theme-variables streak -i home.md -s
```

Then navigate to `localhost:3000` in your browser of choice

---

If you wish to generate a static `html` file, run the following command

```
aglio --theme-variables streak -i home.md -o home.html
```

See the `makedocs` script for usage with `docker run`


## Editing the docs

* Add a markdown file parsable by aglio to the project
* Edit the **makedocs** script to compile the docs to html
* Include the compiled html in our RPM by editing **scripts/prepare-rpm-env.sh**

