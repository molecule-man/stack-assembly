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
* Configuration powered by `Golang Templates <https://golang.org/pkg/text/template/>`_
* Interactive interface which enables user to view, diff and confirm changes to
  be deployed
* Colorized terminal output
* Documentation
* Test coverage


Installation
============

The pre-compiled binaries can be downloaded from `the release page
<https://github.com/molecule-man/stack-assembly/releases>`_. The following OSs
are supported:

* Windows amd64/386
* Linux amd64/386
* Darwin amd64/386

Build it yourself
-----------------

This requires go 1.11 to be installed

.. code-block:: bash

    $ git clone git@github.com:molecule-man/stack-assembly.git
    $ cd stack-assembly
    $ make build

This will build binary inside ``bin`` folder.

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

    The config can be nested:
        stacks:
          parent_tpl:
            name: my-parent-stack
            path: path/to/tpl.json
            stacks:
              child_tpl: # <--- this is ID of the stack we want to deploy
                name: my-child-stack
                path: path/to/tpl.json

    In this case specifying ID of only wanted stack is not enough all the parent IDs
    have to be specified as well:

      stas sync parent_tpl child_tpl

    Usage:
      stas sync [<ID> [<ID> ...]] [flags]

    Aliases:
      sync, deploy

    Flags:
      -h, --help   help for sync

    Global Flags:
      -c, --configs strings   Alternative config file(s). Default: stack-assembly.yaml
      -n, --no-interaction    Do not ask any interactive questions
          --nocolor           Disables color output
      -p, --profile string    AWS named profile (default "default")
      -r, --region string     AWS region

Specifying multiple config files
--------------------------------

You can supply multiple ``-c`` configuration files. When you supply multiple
files, Stack-Assembly combines them into a single configuration. Subsequent
files override and add to their predecessors.

For example, consider this command line:

.. code-block:: bash

    $ stas sync -c stack-assembly.yml -c stack-assembly.staging.yml

The ``stack-assembly.yml`` file might look like this:

.. code-block:: yaml

    stacks:
      ec2machine:
        name: ec2machine-dev
        path: cf-tpls/ec2machine.yml
        parameters:
          Size: t2.micro
          ImageID: ami-rt34fu

And the ``stack-assembly.staging.yml`` file might look like this:

.. code-block:: yaml

    stacks:
      ec2machine:
        name: ec2machine-staging
        parameters:
          Size: t2.medium
        tags:
          ENV: staging

Stack-Assembly will apply configuration from ``stack-assembly.staging.yml`` on
top of ``stack-assembly.yml`` and the result configuration will look like this:

.. code-block:: yaml

    stacks:
      ec2machine:
        name: ec2machine-staging
        path: cf-tpls/ec2machine.yml
        parameters:
          Size: t2.medium
          ImageID: ami-rt34fu
        tags:
          ENV: staging

Configuration
=============

Stack-Assembly uses simple yet powerful config file that can be in one of these
three formats: ``yaml``, ``toml``, ``json``. The next sections will use ``yaml``
as a format.

Config file location
--------------------

Stack-Assembly will firstly try to use file ``stack-assembly.yaml`` in your
project directory. If it's not found then Stack-Assembly will try to use
``stack-assembly.yml``, ``stack-assembly.toml``, ``stack-assembly.json``.

Config file structure
---------------------

Example of Stack-Assembly config file:

.. code-block:: yaml

    settings:
      aws:
        # aws named profile. See the following link for more information
        # https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-profiles.html
        # This configuration option can be overriden by env variable AWS_PROFILE.
        # Or by command line parameter `--profile`
        profile: default

        # aws region. This configuration option can be overriden by env variable
        # AWS_REGION. Or by command line parameter `--region`
        region: us-west-2

    # cloudformation parameters that are global for all stacks
    parameters:
      Env: dev
      ServiceName: myservice

    stacks:
      db:
        # cloudformation stack's name. It's possible to use golang templating
        # inside `name`
        name: "{{ .Params.DbName }}"

        # path to cloudformation template.
        # Either `path` or `body` has to be provided
        path: cf-tpls/rds.yml

        # cloudformation stack's parameters
        parameters:
          Type: db.t2.medium
          # it's possible to use golang templating inside parameter value
          DbName: "{{ .Params.ServiceName }}-{{ .Params.Env }}"

        # cloudformation stack's tags. It's also possible to use golang
        # templating inside tag value
        tags:
          ENV: "{{ .Params.Env }}"

        # it's possible to create a stack policy that will disallow to `update`
        # or `delete` certain stack resources. In this case the policy will be
        # applied to stack resource with `LogicalResourceId` equal to
        # `DbInstance`. See the following link for more information:
        # https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/protect-stack-resources.html
        blocked:
          - DbInstance

      ec2app:
        name: "{{ .Params.ServiceName }}-{{ .Params.Env }}-ec2app"
        parameters:
          Type: t2.micro

        # `path` is not the only way to specify cloudformation template. It's
        # possible to specify the whole template body inside the config. It
        # might be especially useful when template generating tool (as e.g.
        # troposphere) is used.
        # In this example, given that `Env` is equal to "dev", body will have
        # contents of the output produced by executing
        # `python terraform_tpls/ec2.py dev`
        body: |
          {{ .Params.Env | Exec "python" "terraform_tpls/ec2.py" }}

        # dependsOn instruction tells Stack-Assembly that this stack should be
        # deployed after `db` stack is deployed
        dependsOn:
          - db

        # In some cases, you must explicity acknowledge that your stack template
        # contains certain capabilities in order for AWS CloudFormation to
        # create the stack. For more information, see
        # https://docs.aws.amazon.com/AWSCloudFormation/latest/APIReference/API_CreateStack.html
        capabilities:
          - CAPABILITY_IAM

        # Rollback triggers enable you to have AWS CloudFormation monitor the
        # state of your application during stack creation and updating, and to
        # roll back that operation if the application breaches the threshold of
        # any of the alarms you've specified.
        # For more information, see
        # https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/using-cfn-rollback-triggers.html
        rollbackConfiguration:
          monitoringTimeInMinutes: 1
          rollbackTriggers:
            - arn: arn:aws:cloudwatch:{{ .AWS.Region }}:{{ .AWS.AccountID }}:alarm:{{ .Params.ServiceName }}-errors
              type: AWS::CloudWatch::Alarm

Config nesting
--------------

Every stack in stack-assembly config can contain nested stacks. This enables
possibility to deploy subgroup (subtree) of stacks.

.. code-block:: yaml

    stacks:
      staging:

        # settings, as well as parameters, are propagated down the tree.
        # All the child stacks of `staging` inherit settings and parameters
        # defined at `staging` level
        settings:
          aws:
            region: eu-west-1
        parameters:
          Env: staging

        stacks:
          db:
            name: "db-staging"
            path: cf-tpls/rds.yml
          app:
            name: "app-staging"
            path: cf-tpls/app.yml

      production:

        settings:
          aws:
            region: us-east-1
        parameters:
          Env: production

        stacks:
          db:
            name: "db-production"
            path: cf-tpls/rds.yml
          app:
            name: "app-production"
            path: cf-tpls/app.yml

Having this config one can deploy all the stacks under ``production`` by
running:

.. code-block:: bash

    stas sync production

Or, if one needs to deploy ``db`` stack under ``staging``, one can use the
following command:

.. code-block:: bash

    stas sync staging db

Reuse
-----

When writing complex config, it's almost inevitable to have duplication in the
config. This section describes how stack-assembly helps to avoid
copying-and-pasting.

Let's say we have a stack we want to deploy multiple times in different
environments. Each environment is different from each other only by handful of
parameters. Then we put the reused stack under the ``definitions`` in the root
of the config. And then we can (re)use this stack in the config by referencing
this stack with ``$basedOn`` field in the config:

.. code-block:: yaml

    stacks:
      staging:
        "$basedOn": reused_stack
        parameters:
          Env: staging

      production:
        "$basedOn": reused_stack
        parameters:
          Env: production

    definitions:
      reused_stack:
        name: "reused-stack-{{ .Params.Env }}"
        path: cf-tpls/stack.yml

AWS credentials
===============

If you've ever used awscli or similar tool you probably already know about aws
credentials file. Stack-Assembly also uses this file to read credentials. The
default location of this file is ``$HOME/.aws/credentials``. You can find more
information in `AWS documentation
<https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-files.html>`_.

For the sake of example let's consider that you have configured aws credentials
and now have this files in your home folder:

**~/.aws/credentials**

::

    [default]
    aws_access_key_id=AKIAIOSFODNN7EXAMPLE
    aws_secret_access_key=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY

**~/.aws/config**

::

    [default]
    region=us-west-2

Now you have couple of options:

1. Specify profile and region in the config file. See `Config file structure`_:

.. code-block:: yaml

    settings:
      aws:
        profile: default
        region: eu-west-1

2. Use environmental variables:

.. code-block:: bash

    $ export AWS_PROFILE=default
    $ export AWS_REGION=eu-west-1
    $ stas sync

3. Use command line flags:

.. code-block:: bash

    $ stas sync --profile default --region eu-west-1

Other commands
==============

Apart from `sync` command there are also couple of handy other commands you can
use:

.. code-block:: bash

    $ stas help
    Usage:
      stas [command]

    Available Commands:
      delete      Deletes deployed stacks
      diff        Show diff of the stacks to be deployed
      dump-config Dump loaded config into stdout
      help        Help about any command
      info        Show info about the stack
      sync        Synchronize (deploy) stacks

TODO
====

* Enable user to unblock the blocked resource (interactively).
* Github support.
* Add ci.
* Add possibility to introspect aws resources??
