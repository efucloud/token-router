import { history, useModel } from '@umijs/max';
import { Spin } from 'antd';
import { useEffect } from 'react';

const HomePage = () => {
  const { initialState } = useModel('@@initialState');

  useEffect(() => {
    if (initialState?.currentUser) {
      history.replace('/personal/dashboard');
    }
  }, [initialState?.currentUser]);

  return (
    <div className="auth-status">
      <Spin size="small" />
    </div>
  );
};

export default HomePage;
