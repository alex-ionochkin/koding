module.exports = {

    "accessLevel":"private",
    "originId":"57043a0bd37cef1a22a5c427",
    "inUse":false,
    "config": {
      "requiredData": {
        "user":["username"],
        "group":["slug"]
      },
      "requiredProviders":["aws","koding"],
      "verified":true
    },
    "meta":{
      "data":{
        "createdAt":"2016-04-05T22:47:24.542Z",
        "modifiedAt":"2016-04-05T22:47:29.643Z",
        "likes":0
      },
      "createdAt":"2016-04-05T22:47:24.542Z",
      "modifiedAt":"2016-04-05T22:47:29.643Z",
      "likes":0
    },
    "machines":[ {
      "label":"example_1",
      "provider":"aws",
      "region":"us-east-1",
      "source_ami":"ami-cf35f3a4",
      "instanceType":"t2.nano",
      "provisioners":[]
    } ],
    "bongo_":{
      "constructorName":"JStackTemplate",
      "instanceId":"39ae9465233ba116e4c937500b279665"
    },
    "isDefault":false,
    "watchers":{},
    "title":"Default stack template2",
    "template":{
      "content":"{&quot;provider&quot;:{&quot;aws&quot;:{&quot;access_key&quot;:&quot;${var.aws_access_key}&quot;,&quot;secret_key&quot;:&quot;${var.aws_secret_key}&quot;}},&quot;resource&quot;:{&quot;aws_instance&quot;:{&quot;example_1&quot;:{&quot;instance_type&quot;:&quot;t2.nano&quot;,&quot;ami&quot;:&quot;&quot;,&quot;tags&quot;:{&quot;Name&quot;:&quot;${var.koding_user_username}-${var.koding_group_slug}&quot;}}}}}",
      "details":{"lastUpdaterId":"57043a0bd37cef1a22a5c427"},
      "rawContent":"# Here is your stack preview\n# You can make advanced changes like modifying your VM,\n# installing packages, and running shell commands.\n\nprovider:\n  aws:\n    access_key: &#39;${var.aws_access_key}&#39;\n    secret_key: &#39;${var.aws_secret_key}&#39;\nresource:\n  aws_instance:\n    example_1:\n      instance_type: t2.nano\n      ami: &#39;&#39;\n      tags:\n        Name: &#39;${var.koding_user_username}-${var.koding_group_slug}&#39;\n",
      "sum":"bf5fa3f7c889f40ea79e82e0363f34d8de76c9db"
    },
    "_id":"5704407c8e15dd3674060102",
    "credentials":{
      "aws":["f1e165b7aa83885c935523c1e75f9403"]
    },
    "description":"##### Readme text for this stack template\n\nYou can write down a readme text for new users.\nThis text will be shown when they want to use this stack.\nYou can use markdown with the readme content.\n\n","group":"kiskis"
}