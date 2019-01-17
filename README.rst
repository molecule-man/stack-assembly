Stack-Assembly
##############

Stack-Assembly is a command line tool to configure and deploy AWS Cloudformation
stacks in a safe way. The safety aspect is enabled by utilizing Cloudformation
`Changesets
<https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/using-cfn-updating-stacks-changesets.html>`_
and letting the user to view and confirm changes to be deployed.


.. contents::

.. section-numbering::

Main features
=============

* No dependencies (NodeJS, Python interpreter, aws cli etc.) - Stack-Assembly is
  a single statically linked binary
* Powerful configuration
* Powered by `Golang Templates <https://golang.org/pkg/text/template/>`_
* Interactive interface which enables user to view, diff and confirm changes to
  be deployed
* Colorized terminal output
* Documentation
* Test coverage


Installation
============

To be added

Quick start example
===================

For demonstration purposes it is assumed that there exists file
``./path/to/cf-tpls/sqs.yaml`` containing cloudformation template you
want to deploy. For example:

.. code-block:: yaml

    AWSTemplateFormatVersion: "2010-09-09"
    Parameters:
      QueueName:
        Type: String
      VisibilityTimeout:
        Type: Number
    Resources:
      MyQueue:
        Type: AWS::SQS::Queue
        Properties:
          QueueName: !Ref QueueName
          VisibilityTimeout: !Ref VisibilityTimeout

Then create Stack-Assembly configuration file in the root folder of your project
``stack-assembly.yaml``:

.. code-block:: yaml

    # In this simple example two stacks are configured. Both stacks use the same
    # cloudformation template
    stacks:
      # tpl1 is the id of the stack. This id has meaning only inside this
      # config. You can use this id to deploy a particular stack instead of
      # deploying all stacks as it's done by default. You can do it by running
      # `stas sync tpl1`
      tpl1:
        name: demo-tpl1
        path: path/to/cf-tpls/sqs.yaml
        # parameters is a key-value map where values are strings. Numeric
        # parameters have to be defined as strings as you can see in the example
        # of VisibilityTimeout parameter
        parameters:
          QueueName: demo1
          VisibilityTimeout: "10"
      tpl2:
        name: demo-tpl2
        path: path/to/cf-tpls/sqs.yaml
        parameters:
          QueueName: demo2
          VisibilityTimeout: "20"

Assuming you have configured `AWS credentials`_ then you can deploy your stacks
by running:

.. code-block:: bash

    $ stas sync

By default Stack-Assembly is executed in interactive mode. During the deployment
it shows the changes that are about to be deployed and asks user's confirmation
to proceed with deployment.

Usage
=====

.. code-block::

    $ stas help sync
    Creates or updates stacks specified in the config file(s).

    By default sync command deploys all the stacks described in the config file(s).
    To deploy a particular stack, ID argument has to be provided. ID is an
    identifier of a stack within the config file. For example, ID is tpl1 in the
    following yaml config:

    	stacks:
    	  tpl1: # <--- this is ID
    		name: mystack
    		path: path/to/tpl.json

    Usage:
      stas sync [ID] [flags]

    Aliases:
      sync, deploy

    Flags:
      -h, --help             help for sync
      -n, --no-interaction   Do not ask any interactive questions

    Global Flags:
      -c, --configs strings   Stack-Assembly config files
    	  --nocolor           Disables color output<Paste>

Configuration
=============

To be added

AWS credentials
===============

To be added

TODO
====

* Add possibility to introspect aws resources.
* Enable user to unblock the blocked resource (interactively).
* Github support.
* Add ci.
