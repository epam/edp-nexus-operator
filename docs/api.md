# API Reference

Packages:

- [edp.epam.com/v1alpha1](#edpepamcomv1alpha1)

# edp.epam.com/v1alpha1

Resource Types:

- [Nexus](#nexus)

- [NexusRole](#nexusrole)

- [NexusUser](#nexususer)




## Nexus
<sup><sup>[↩ Parent](#edpepamcomv1alpha1 )</sup></sup>






Nexus is the Schema for the nexus API.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>edp.epam.com/v1alpha1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>Nexus</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#nexusspec">spec</a></b></td>
        <td>object</td>
        <td>
          NexusSpec defines the desired state of Nexus.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#nexusstatus">status</a></b></td>
        <td>object</td>
        <td>
          NexusStatus defines the observed state of Nexus.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### Nexus.spec
<sup><sup>[↩ Parent](#nexus)</sup></sup>



NexusSpec defines the desired state of Nexus.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>secret</b></td>
        <td>string</td>
        <td>
          Secret is the name of the k8s object Secret related to nexus. Secret should contain a user field with a nexus username and a password field with a nexus password.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>url</b></td>
        <td>string</td>
        <td>
          Url is the url of nexus instance.<br/>
        </td>
        <td>true</td>
      </tr></tbody>
</table>


### Nexus.status
<sup><sup>[↩ Parent](#nexus)</sup></sup>



NexusStatus defines the observed state of Nexus.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>connected</b></td>
        <td>boolean</td>
        <td>
          Connected shows if operator is connected to nexus.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>error</b></td>
        <td>string</td>
        <td>
          Error represents error message if something went wrong.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

## NexusRole
<sup><sup>[↩ Parent](#edpepamcomv1alpha1 )</sup></sup>






NexusRole is the Schema for the nexusroles API.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>edp.epam.com/v1alpha1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>NexusRole</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#nexusrolespec">spec</a></b></td>
        <td>object</td>
        <td>
          NexusRoleSpec defines the desired state of NexusRole.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#nexusrolestatus">status</a></b></td>
        <td>object</td>
        <td>
          NexusRoleStatus defines the observed state of NexusRole.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### NexusRole.spec
<sup><sup>[↩ Parent](#nexusrole)</sup></sup>



NexusRoleSpec defines the desired state of NexusRole.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>id</b></td>
        <td>string</td>
        <td>
          ID is the id of the role. ID should be unique across all roles.<br/>
          <br/>
            <i>Validations</i>:<li>self == oldSelf: Value is immutable</li>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name is the name of the role.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#nexusrolespecnexusref">nexusRef</a></b></td>
        <td>object</td>
        <td>
          NexusRef is a reference to Nexus custom resource.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>description</b></td>
        <td>string</td>
        <td>
          Description of nexus role.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>privileges</b></td>
        <td>[]string</td>
        <td>
          Privileges is a list of privileges assigned to role.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### NexusRole.spec.nexusRef
<sup><sup>[↩ Parent](#nexusrolespec)</sup></sup>



NexusRef is a reference to Nexus custom resource.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name specifies the name of the Nexus resource.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>kind</b></td>
        <td>string</td>
        <td>
          Kind specifies the kind of the Nexus resource.<br/>
          <br/>
            <i>Default</i>: Nexus<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### NexusRole.status
<sup><sup>[↩ Parent](#nexusrole)</sup></sup>



NexusRoleStatus defines the observed state of NexusRole.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>error</b></td>
        <td>string</td>
        <td>
          Error is an error message if something went wrong.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          Value is a status of the role.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>

## NexusUser
<sup><sup>[↩ Parent](#edpepamcomv1alpha1 )</sup></sup>






NexusUser is the Schema for the nexususers API.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>edp.epam.com/v1alpha1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>NexusUser</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#nexususerspec">spec</a></b></td>
        <td>object</td>
        <td>
          NexusUserSpec defines the desired state of NexusUser.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#nexususerstatus">status</a></b></td>
        <td>object</td>
        <td>
          NexusUserStatus defines the observed state of NexusUser.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### NexusUser.spec
<sup><sup>[↩ Parent](#nexususer)</sup></sup>



NexusUserSpec defines the desired state of NexusUser.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>email</b></td>
        <td>string</td>
        <td>
          Email is the email address of the user.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>firstName</b></td>
        <td>string</td>
        <td>
          FirstName of the user.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>id</b></td>
        <td>string</td>
        <td>
          ID is the username of the user. ID should be unique across all users.<br/>
          <br/>
            <i>Validations</i>:<li>self == oldSelf: Value is immutable</li>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>lastName</b></td>
        <td>string</td>
        <td>
          LastName of the user.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#nexususerspecnexusref">nexusRef</a></b></td>
        <td>object</td>
        <td>
          NexusRef is a reference to Nexus custom resource.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>roles</b></td>
        <td>[]string</td>
        <td>
          Roles is a list of roles assigned to user.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>secret</b></td>
        <td>string</td>
        <td>
          Secret is the reference of the k8s object Secret for the user password. Format: $secret-name:secret-key. Updating user password is not supported.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>status</b></td>
        <td>enum</td>
        <td>
          Status is a status of the user.<br/>
          <br/>
            <i>Enum</i>: active, disabled<br/>
            <i>Default</i>: active<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### NexusUser.spec.nexusRef
<sup><sup>[↩ Parent](#nexususerspec)</sup></sup>



NexusRef is a reference to Nexus custom resource.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>
          Name specifies the name of the Nexus resource.<br/>
        </td>
        <td>true</td>
      </tr><tr>
        <td><b>kind</b></td>
        <td>string</td>
        <td>
          Kind specifies the kind of the Nexus resource.<br/>
          <br/>
            <i>Default</i>: Nexus<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>


### NexusUser.status
<sup><sup>[↩ Parent](#nexususer)</sup></sup>



NexusUserStatus defines the observed state of NexusUser.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>error</b></td>
        <td>string</td>
        <td>
          Error is an error message if something went wrong.<br/>
        </td>
        <td>false</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>
          Value is a status of the user.<br/>
        </td>
        <td>false</td>
      </tr></tbody>
</table>
