Feature: stas sync with parameters

    @short
    Scenario: sync single valid template with parameters
        Given file "cfg.yaml" exists:
            """
            parameters:
              Env: dev
              Cluster1: cluster1-%scenarioid%
            stacks:
              stack1:
                name: stastest-param-%scenarioid%
                path: tpls/stack1.yml
                parameters:
                  Cluster2: cluster2-%scenarioid%
                tags:
                  STAS_TEST: '%featureid%'
            """
        And file "tpls/stack1.yml" exists:
            """
            Parameters:
              Env:
                Type: String
              Cluster1:
                Type: String
              Cluster2:
                Type: String
            Resources:
              EcsCluster1:
                Type: AWS::ECS::Cluster
                Properties:
                  ClusterName: !Sub "${Cluster1}-${Env}"
              EcsCluster2:
                Type: AWS::ECS::Cluster
                Properties:
                  ClusterName: !Ref Cluster2
            """
        When I successfully run "sync -c cfg.yaml --no-interaction"
        Then stack "stastest-param-%scenarioid%" should have status "CREATE_COMPLETE"

    Scenario: sync doesn't fail when I remove parameter from config. Old parameter value is used
        Given file "cfg.yaml" exists:
            """
            parameters:
              Env: dev
              Password: mysecretpassword
            stacks:
              stack1:
                name: stastest-rmparam-%scenarioid%
                path: tpls/stack1.yml
                tags:
                  STAS_TEST: '%featureid%'
            """
        And file "tpls/stack1.yml" exists:
            """
            Parameters:
              Env:
                Type: String
              Password:
                Type: String
                NoEcho: true

            Resources:
              MyeSecret:
                Type: 'AWS::SecretsManager::Secret'
                Properties:
                  Name: !Sub "${AWS::StackName}-secret-${Env}"
                  SecretString: !Sub '{"password":"${Password}"}'
            """
        Given I successfully run "sync -c cfg.yaml --no-interaction"
        When I modify file "cfg.yaml":
            """
            parameters:
              Env: prod
            stacks:
              stack1:
                name: stastest-rmparam-%scenarioid%
                path: tpls/stack1.yml
                tags:
                  STAS_TEST: '%featureid%'
            """
        And I successfully run "sync -c cfg.yaml --no-interaction"
        Then stack "stastest-rmparam-%scenarioid%" should have status "UPDATE_COMPLETE"

    @short
    Scenario: sync prompts me to enter parameter value if it's not present in config when I create stack
        Given file "cfg.yaml" exists:
            """
            stacks:
              stack1:
                name: stastest-%scenarioid%
                path: tpls/stack1.yml
                tags:
                  STAS_TEST: '%featureid%'
            """
        And file "tpls/stack1.yml" exists:
            """
            Parameters:
              User:
                Type: String
              Password:
                Type: String
                NoEcho: true

            Resources:
              MyeSecret:
                Type: 'AWS::SecretsManager::Secret'
                Properties:
                  Name: !Sub "${AWS::StackName}-secret"
                  SecretString: !Sub '{"user": "${User}", "password":"${Password}"}'
            """
        Given I launched "sync -c cfg.yaml"
        And terminal shows:
            """
            the following parameters are required but not provided: User, Password
            Enter User:
            """
        When I enter "myuser"
        Then terminal shows:
            """
            Enter Password:
            """
        When I enter "mysecret"
        Then terminal shows:
            """
            What now>
            """
        When I enter "s"
        Then launched program should exit with zero status
        And stack "stastest-%scenarioid%" should have status "CREATE_COMPLETE"

    Scenario: sync prompts me to enter parameter value if it's not present in config when I update stack
        Given file "cfg.yaml" exists:
            """
            stacks:
              stack1:
                name: stastest-%scenarioid%
                path: tpls/stack1.yml
                tags:
                  STAS_TEST: '%featureid%'
            """
        And file "tpls/stack1.yml" exists:
            """
            Resources:
              EcsCluster:
                Type: AWS::ECS::Cluster
                Properties:
                  ClusterName: !Sub "${AWS::StackName}"
            """
        And I successfully run "sync -c cfg.yaml --no-interaction"
        When I modify file "tpls/stack1.yml":
            """
            Parameters:
              Env:
                Type: String
            Resources:
              EcsCluster:
                Type: AWS::ECS::Cluster
                Properties:
                  ClusterName: !Sub "${AWS::StackName}"
            """
        And I launched "sync -c cfg.yaml"
        Then terminal shows:
            """
            the following parameters are required but not provided: Env
            Enter Env:
            """

    @short
    Scenario: sync doesn't promt when parameter with default value is missing
        Given file "cfg.yaml" exists:
            """
            stacks:
              stack1:
                name: stastest-defaultparam-%scenarioid%
                path: tpls/stack1.yml
                tags:
                  STAS_TEST: '%featureid%'
            """
        And file "tpls/stack1.yml" exists:
            """
            Parameters:
              Env:
                Type: String
                Default: dev

            Resources:
              EcsCluster:
                Type: AWS::ECS::Cluster
                Properties:
                  ClusterName: !Sub "${AWS::StackName}-${Env}"
            """
        Then I successfully run "sync -c cfg.yaml --no-interaction"
        And stack "stastest-defaultparam-%scenarioid%" should have status "CREATE_COMPLETE"

    @short
    Scenario: parameters can be taken from command line
        Given file "cfg.yaml" exists:
            """
            stacks:
              stack1:
                name: stastest-1-%scenarioid%
                path: tpls/stack.yml
            """
        And file "tpls/stack.yml" exists:
            """
            Resources:
              EcsCluster:
                Type: AWS::ECS::Cluster
                Properties:
                  ClusterName: !Sub "${AWS::StackName}"
            """
        When I successfully run "dump-config -c cfg.yaml -f json -v param1=Foo -v Param2=bar"
        Then node "Parameters.param1" in json output should be:
            """
            "Foo"
            """
        And node "Parameters.Param2" in json output should be:
            """
            "bar"
            """
