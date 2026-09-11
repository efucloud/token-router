import { defineConfig } from 'umi';
import defaultSettings from './defaultSettings';
import proxy from './proxy';
import routes from './routes';

const { UMI_ENV = 'dev' } = process.env;

export default defineConfig({
  antd: {
    configProvider: {
      theme: {
        token: {
          fontFamily:
            "AlibabaSans, -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif",
        },
      },
    },
  },
  fastRefresh: true,
  esbuildMinifyIIFE: true,
  hash: true,
  initialState: {},
  access: {},
  layout: {
    locale: true,
    ...defaultSettings,
  },
  locale: {
    default: 'zh-CN',
    antd: true,
    baseNavigator: true,
  },
  model: {},
  proxy: proxy[UMI_ENV as keyof typeof proxy],
  request: {},
  routes,
  title: 'Token Router',
});
