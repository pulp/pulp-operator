Contributing
============

Pull Request Checklist
------------------------
1. Make sure your change does not break idempotency tests. See [Testing](#Testing)
(or let CI run the tests for you if you are certain it is idempotent.)
If a task cannot be made idempotent, add the tag [molecule-idempotence-notest](https://github.com/ansible-community/molecule/issues/816#issuecomment-573319053).
2. Unless a change is small or doesn't affect users, create an issue on
[github](https://github.com/pulp/pulp-operator/issues/new).
3. Add [a changelog update.](https://docs.pulpproject.org/contributing/git.html#changelog-update)
4. Write an excellent [Commit Message.](https://docs.pulpproject.org/contributing/git.html#commit-message)
Make sure you reference and link to the issue.
5. Push your branch to your fork and open a [Pull request across forks.](https://help.github.com/articles/creating-a-pull-request-from-a-fork/)
6. Add GitHub labels as appropriate.

Testing
-------

The tests can be run as they are on travis with **tox**, or they can run with various options using
**molecule** directly.

**Requirements:**
Install Docker, and add yourself to the group that is authorized to
administer containers, and log out and back in to make the permissions change
take effect. The authorized group is typically the "docker" group:

```bash
gpasswd --add "$(whoami)" docker
```

**NOTE:** Docker containers can differ from bare-metal or VM OS installs.
They can have different packages installed, they can run different kernels,
and so on.

**Using Molecule:**

1. Install [molecule](https://molecule.readthedocs.io/en/latest/),
[molecule-inspec](https://github.com/ansible-community/molecule-inspec),
and [ansible-lint](https://docs.ansible.com/ansible-lint/).
It is recommended that you do so with `pip` in a virtualenv.

2. Run molecule commands.

      - Test all scenarios on all hosts.
      ```bash
      molecule test --all
      ```

      - Test a specific scenario.
      ```bash
      molecule test --scenario-name test-local
      ```

      - Use debug for increased verbosity.
      ```bash
      molecule --debug test --all
      ```

      - Create and provision, but don't run tests or destroy.
      ```bash
      molecule converge --all
      ```

Docs Testing
------------

On Fedora:
```
sudo dnf install mkdocs python3-pymdown-extensions
```

Cross-platform:
```
pip install mkdocs pymdown-extensions
```

Then:
```
mkdocs serve
```
Click the link it outputs. As you save changes to files modified in your editor,
the browser will automatically show the new content.
