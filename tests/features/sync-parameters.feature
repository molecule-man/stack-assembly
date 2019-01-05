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

    @wip
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
