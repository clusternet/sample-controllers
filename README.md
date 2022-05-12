# Sample controller

This is a controller for user-defined processing logic.

This sample is a feedinventory controller, if you want use Dividing scheduler strategy, and the workload is not kubernetes nativ type , you can use this controller after development.

## Install Controller
1. set hub command `FeedInventory=false`
2. kubectl apply -f script/sample-controller.yaml