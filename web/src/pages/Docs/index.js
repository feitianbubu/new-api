import React, { useContext } from 'react';
import { Card } from '@douyinfe/semi-ui';
import { StatusContext } from '../../context/Status';

const Docs = () => {
  const [statusState] = useContext(StatusContext);
  const docsLink = statusState?.status?.docs_link || localStorage.getItem('docs_link') || 'https://dev.clinx.work/swag.html';

  return (
    <Card
      bodyStyle={{ padding: 0, height: 'calc(100vh - 56px)' }}
      style={{ height: '100%', minHeight: '600px', boxShadow: 'none', background: 'transparent' }}
      bordered={false}
    >
      <iframe
        src={docsLink}
        style={{ width: '100%', height: '100%', border: 'none', minHeight: '600px' }}
        title='API 文档'
        sandbox='allow-same-origin allow-scripts allow-popups allow-forms'
      />
    </Card>
  );
};

export default Docs; 