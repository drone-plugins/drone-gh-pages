local pipeline = import 'pipeline.libsonnet';
local name = 'drone-gh-pages';

[
  pipeline.test('linux', 'amd64'),
  pipeline.build(name, 'linux', 'amd64'),
  pipeline.build(name, 'linux', 'arm64'),
  pipeline.build(name, 'linux', 'arm'),
  pipeline.notifications(depends_on=[
    'linux-amd64',
    'linux-arm64',
    'linux-arm',
  ]),
]
