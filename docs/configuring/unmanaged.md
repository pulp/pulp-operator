# Disabling the Operator

Sometimes we need to stop the Operator reconciliation process.  
An example of such situation is when troubleshooting and would like
to test different configurations without modifying Pulp CR.

Setting the Operator as `unmanaged` will stop all Operator's tasks, which means no
more reconciliation and/or reprovisioning of objects. It is like disabling the Operator.

!!! note
    Pulp cluster will keep running even if the Operator is `unmanaged`, but no new modification
    done on Pulp CR will reflect in Pulp objects.


## Setting the Operator as unmanaged

To set the Operator as unmanaged update Pulp CR:
```
...
spec:
  unmanaged: true
...
```

From now on:

* any modification done on [objects managed](/pulp_operator/faq/#which-resources-are-managed-by-the-operator) by the Operator will not be reconciled
* any removed object - [managed](/pulp_operator/faq/#which-resources-are-managed-by-the-operator) by the Operator -  will not be reprovisioned



## Setting the Operator back as managed

To set the Operator as managed again, just update Pulp CR:
```
...
spec:
  unmanaged: false
...
```

It is also possible to delete the `unmanaged` field, since the default value is false.

!!! Warning
    Putting back the Operator in a `managed` state will:

    * **overwrite** any modifications done while it was `unmanaged`
    * **reprovision** any object deleted while it was `unmanaged`
