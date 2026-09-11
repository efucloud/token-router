const target = process.env.API_PROXY_TARGET || 'http://localhost:9006';

export default {
  dev: {
    '/api/': {
      changeOrigin: true,
      target,
    },
    '/v1/': {
      changeOrigin: true,
      target,
    },
  },
};
