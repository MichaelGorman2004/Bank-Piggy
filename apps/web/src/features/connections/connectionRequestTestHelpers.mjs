/** Gives a test direct control over request completion without timers. */
export function deferred() {
  let resolve;
  let reject;
  const promise = new Promise((accept, decline) => {
    resolve = accept;
    reject = decline;
  });
  return { promise, resolve, reject };
}

/** Records visible connection state and shared request ownership independently of React. */
export function ownerFixture() {
  const state = { busy: false, loading: false, error: null, feedback: null, data: null };
  const options = {
    generation: { current: 0 },
    mounted: { current: true },
    lock: { current: false },
    pendingRead: { current: false },
    reads: { current: { active: null, queued: null } },
    read: async () => {},
    setBusy: (busy) => {
      state.busy = busy;
    },
    setLoading: (loading) => {
      state.loading = loading;
    },
    setError: (error) => {
      state.error = error;
    },
    setFeedback: (feedback) => {
      state.feedback = feedback;
    },
    setData: (data) => {
      state.data = data;
    },
  };
  return { state, options };
}
