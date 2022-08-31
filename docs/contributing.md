Contributing
============

Pull Request Checklist
------------------------
1. Unless a change is small or doesn't affect users, create an issue on
[github](https://github.com/pulp/pulp-operator/issues/new).
2. Add [a changelog update.](https://docs.pulpproject.org/contributing/git.html#changelog-update)
3. Write an excellent [Commit Message.](https://docs.pulpproject.org/contributing/git.html#commit-message)
Make sure you reference and link to the issue.
4. Push your branch to your fork and open a [Pull request across forks.](https://help.github.com/articles/creating-a-pull-request-from-a-fork/)
5. Add GitHub labels as appropriate.

Testing
-------

The tests can be run ...

**Requirements:**
Install Docker, and add yourself to the group that is authorized to
administer containers, and log out and back in to make the permissions change
take effect. The authorized group is typically the "docker" group:

```bash
gpasswd --add "$(whoami)" docker
```

Docs Testing
------------

Cross-platform:
```
pip install mkdocs pymdown-extensions mkdocs-material mike mkdocs-git-revision-date-plugin
```

Then:
```
mkdocs serve
```
Click the link it outputs. As you save changes to files modified in your editor,
the browser will automatically show the new content.


Debugging
---------

Debugging ...
