import type {NextConfig} from "next";

const isExport = process.env.NEXT_STANDALONE_EXPORT === 'true';

const nextConfig: NextConfig = {
  reactCompiler: true,
  experimental: {
  },
  ...(isExport ? { output: 'export' } : {
    async rewrites() {
      const backendUrl = process.env.WAVELET_BACKEND_URL || 'http://localhost:8000';
      return [
        // 易支付兼容接口 - 创建订单
        {
          source: '/epay/pay/:path*',
          destination: `${ backendUrl }/pay/:path*`,
        },
        // 易支付兼容接口 - 查询订单和退款
        {
          source: '/epay/api.php',
          destination: `${ backendUrl }/api.php`,
        },
        // Credit 协议接口 - 商户分发
        {
          source: '/lpay/distribute',
          destination: `${ backendUrl }/pay/distribute`,
        },
        // 上传文件静态资源
        {
          source: '/f/:id',
          destination: `${ backendUrl }/f/:id`,
        },
        // 标准 RESTful API 接口
        {
          source: '/api/:path*',
          destination: `${ backendUrl }/api/:path*`,
        }
      ];
    },
  })
};

export default nextConfig;
