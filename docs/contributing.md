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

The tests can be run with the following command:
```bash
make test
```

If you want to run the tests inside your editor/IDE, you may need download the required binaries,
you can do it by running:
```bash
make testbin
```

Docs Testing
------------

Cross-platform:
```
pip install -r docs/doc_requirements.txt
```

Then:
```
mkdocs serve
```
Click the link it outputs. As you save changes to files modified in your editor,
the browser will automatically show the new content.


Debugging
---------

1. Ensure you have a cluster
  ```bash
  minikube start --vm-driver=docker --extra-config=apiserver.service-node-port-range=80-32000
  ```
2. Build and apply the manifests
  ```bash
  make local
  ```
3. Apply your custom resource
  ```bash
  kubectl apply -f config/samples/simple.yaml
  ```

The following steps are biased towards [vscode](https://code.visualstudio.com/):

1. Make sure you have the [Go extension](https://marketplace.visualstudio.com/items?itemName=golang.Go) installed
2. Make sure you have a `.vscode/launch.json` file with at least this config:
  ```json
    {
        "version": "0.2.0",
        "configurations": [
            {
            "name": "Launch Operator",
            "type": "go",
            "request": "launch",
            "mode": "auto",
            "program": "${workspaceFolder}"
            }
        ]
    }
  ```
  You can learn more about debugging settings [here](https://github.com/golang/vscode-go/wiki/debugging)
