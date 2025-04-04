## Creator battle load test

- We have 2 deployments -- `livekit-loadtest-battle-1` and `livekit-loadtest-battle-2` in {namespace, cluster} = {`live-kit`, `moj-s-oci-live-services-01`}
- First, we'll spawn `targetPods` in both deployments.
- Now, each pod can handle `roomsPerPod`. We can control this by editing each deployment.
- So, total rooms (T) = `targetPods`*`roomsPerPod` in each deployment
- Now, we'll pair 1:1 each of the T rooms in both the deployments for battle by calling `livestream-core-service`.