/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import React, { useContext, useRef, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Button, Form, Icon, Modal } from '@douyinfe/semi-ui';
import { IconGithubLogo } from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import TelegramLoginButton from 'react-telegram-login';
import {
  API,
  showError,
  showInfo,
  showSuccess,
  updateAPI,
  setUserData,
  onGitHubOAuthClicked,
  onOIDCClicked,
  onLinuxDOOAuthClicked,
} from '../../helpers';
import { UserContext } from '../../context/User';
import OIDCIcon from '../common/logo/OIDCIcon';
import WeChatIcon from '../common/logo/WeChatIcon';
import LinuxDoIcon from '../common/logo/LinuxDoIcon';

/**
 * Shared OAuth buttons component for login/register pages
 * @param {Object} props
 * @param {string} props.loginChallenge - OAuth login challenge (optional, for OAuth provider flow)
 * @param {Function} props.onSuccess - Callback when OAuth login succeeds (optional)
 * @param {boolean} props.requireTermsAgreement - Whether to check terms agreement before OAuth
 * @param {boolean} props.agreedToTerms - Whether user has agreed to terms
 */
const OAuthButtons = ({
  loginChallenge = null,
  onSuccess = null,
  requireTermsAgreement = false,
  agreedToTerms = true,
}) => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [, userDispatch] = useContext(UserContext);

  const [showWeChatLoginModal, setShowWeChatLoginModal] = useState(false);
  const [wechatVerificationCode, setWechatVerificationCode] = useState('');
  const [wechatLoading, setWechatLoading] = useState(false);
  const [wechatCodeSubmitLoading, setWechatCodeSubmitLoading] = useState(false);
  const [githubLoading, setGithubLoading] = useState(false);
  const [oidcLoading, setOidcLoading] = useState(false);
  const [linuxdoLoading, setLinuxdoLoading] = useState(false);
  const [githubButtonText, setGithubButtonText] = useState('使用 GitHub 继续');
  const [githubButtonDisabled, setGithubButtonDisabled] = useState(false);
  const githubTimeoutRef = useRef(null);

  const [status] = useState(() => {
    const savedStatus = localStorage.getItem('status');
    return savedStatus ? JSON.parse(savedStatus) : {};
  });

  // Check if any OAuth provider is enabled
  const hasOAuthProviders =
    status.github_oauth ||
    status.oidc_enabled ||
    status.wechat_login ||
    status.linuxdo_oauth ||
    status.telegram_oauth;

  // Check terms agreement
  const checkTermsAgreement = () => {
    if (requireTermsAgreement && !agreedToTerms) {
      showInfo(t('请先阅读并同意用户协议和隐私政策'));
      return false;
    }
    return true;
  };

  // Handle successful login
  const handleLoginSuccess = (data) => {
    userDispatch({ type: 'login', payload: data });
    localStorage.setItem('user', JSON.stringify(data));
    setUserData(data);
    updateAPI();
    showSuccess(t('登录成功！'));

    if (onSuccess) {
      onSuccess(data);
    } else if (loginChallenge) {
      // For OAuth provider flow, redirect to continue
      navigate(`/oauth/login?login_challenge=${loginChallenge}`);
    } else {
      navigate('/console');
    }
  };

  // WeChat login
  const onWeChatLoginClicked = () => {
    if (!checkTermsAgreement()) return;
    setWechatLoading(true);
    setShowWeChatLoginModal(true);
    setWechatLoading(false);
  };

  const onSubmitWeChatVerificationCode = async () => {
    setWechatCodeSubmitLoading(true);
    try {
      const res = await API.get(`/api/oauth/wechat?code=${wechatVerificationCode}`);
      const { success, message, data } = res.data;
      if (success) {
        setShowWeChatLoginModal(false);
        handleLoginSuccess(data);
      } else {
        showError(message);
      }
    } catch (error) {
      showError(t('登录失败，请重试'));
    } finally {
      setWechatCodeSubmitLoading(false);
    }
  };

  // GitHub login
  const handleGitHubClick = () => {
    if (!checkTermsAgreement()) return;
    if (githubButtonDisabled) return;

    setGithubLoading(true);
    setGithubButtonDisabled(true);
    setGithubButtonText(t('正在跳转 GitHub...'));

    if (githubTimeoutRef.current) {
      clearTimeout(githubTimeoutRef.current);
    }
    githubTimeoutRef.current = setTimeout(() => {
      setGithubLoading(false);
      setGithubButtonText(t('请求超时，请刷新页面后重新发起 GitHub 登录'));
      setGithubButtonDisabled(true);
    }, 20000);

    try {
      onGitHubOAuthClicked(status.github_client_id, loginChallenge);
    } finally {
      setTimeout(() => setGithubLoading(false), 3000);
    }
  };

  // OIDC login
  const handleOIDCClick = () => {
    if (!checkTermsAgreement()) return;
    setOidcLoading(true);
    try {
      onOIDCClicked(status.oidc_authorization_endpoint, status.oidc_client_id, false, loginChallenge);
    } finally {
      setTimeout(() => setOidcLoading(false), 3000);
    }
  };

  // LinuxDO login
  const handleLinuxDOClick = () => {
    if (!checkTermsAgreement()) return;
    setLinuxdoLoading(true);
    try {
      onLinuxDOOAuthClicked(status.linuxdo_client_id, loginChallenge);
    } finally {
      setTimeout(() => setLinuxdoLoading(false), 3000);
    }
  };

  // Telegram login
  const onTelegramLoginClicked = async (response) => {
    if (!checkTermsAgreement()) return;

    const fields = ['id', 'first_name', 'last_name', 'username', 'photo_url', 'auth_date', 'hash', 'lang'];
    const params = {};
    fields.forEach((field) => {
      if (response[field]) {
        params[field] = response[field];
      }
    });

    try {
      const res = await API.get('/api/oauth/telegram/login', { params });
      const { success, message, data } = res.data;
      if (success) {
        handleLoginSuccess(data);
      } else {
        showError(message);
      }
    } catch (error) {
      showError(t('登录失败，请重试'));
    }
  };

  // Cleanup on unmount
  React.useEffect(() => {
    return () => {
      if (githubTimeoutRef.current) {
        clearTimeout(githubTimeoutRef.current);
      }
    };
  }, []);

  if (!hasOAuthProviders) {
    return null;
  }

  return (
    <>
      <div className='space-y-3'>
        {status.wechat_login && (
          <Button
            theme='outline'
            className='w-full h-12 flex items-center justify-center !rounded-full border border-gray-200 hover:bg-gray-50 transition-colors'
            type='tertiary'
            icon={<Icon svg={<WeChatIcon />} style={{ color: '#07C160' }} />}
            onClick={onWeChatLoginClicked}
            loading={wechatLoading}
          >
            <span className='ml-3'>{t('使用 微信 继续')}</span>
          </Button>
        )}

        {status.github_oauth && (
          <Button
            theme='outline'
            className='w-full h-12 flex items-center justify-center !rounded-full border border-gray-200 hover:bg-gray-50 transition-colors'
            type='tertiary'
            icon={<IconGithubLogo size='large' />}
            onClick={handleGitHubClick}
            loading={githubLoading}
            disabled={githubButtonDisabled}
          >
            <span className='ml-3'>{githubButtonText}</span>
          </Button>
        )}

        {status.oidc_enabled && (
          <Button
            theme='outline'
            className='w-full h-12 flex items-center justify-center !rounded-full border border-gray-200 hover:bg-gray-50 transition-colors'
            type='tertiary'
            icon={<OIDCIcon style={{ color: '#1877F2' }} />}
            onClick={handleOIDCClick}
            loading={oidcLoading}
          >
            <span className='ml-3'>{t('使用 OIDC 继续')}</span>
          </Button>
        )}

        {status.linuxdo_oauth && (
          <Button
            theme='outline'
            className='w-full h-12 flex items-center justify-center !rounded-full border border-gray-200 hover:bg-gray-50 transition-colors'
            type='tertiary'
            icon={<LinuxDoIcon style={{ color: '#E95420', width: '20px', height: '20px' }} />}
            onClick={handleLinuxDOClick}
            loading={linuxdoLoading}
          >
            <span className='ml-3'>{t('使用 LinuxDO 继续')}</span>
          </Button>
        )}

        {status.telegram_oauth && (
          <div className='flex justify-center my-2'>
            <TelegramLoginButton
              dataOnauth={onTelegramLoginClicked}
              botName={status.telegram_bot_name}
            />
          </div>
        )}
      </div>

      {/* WeChat Login Modal */}
      <Modal
        title={t('微信扫码登录')}
        visible={showWeChatLoginModal}
        maskClosable={true}
        onOk={onSubmitWeChatVerificationCode}
        onCancel={() => setShowWeChatLoginModal(false)}
        okText={t('登录')}
        centered={true}
        okButtonProps={{ loading: wechatCodeSubmitLoading }}
      >
        <div className='flex flex-col items-center'>
          <img src={status.wechat_qrcode} alt='微信二维码' className='mb-4' />
        </div>
        <div className='text-center mb-4'>
          <p>{t('微信扫码关注公众号，输入「验证码」获取验证码（三分钟内有效）')}</p>
        </div>
        <Form>
          <Form.Input
            field='wechat_verification_code'
            placeholder={t('验证码')}
            label={t('验证码')}
            value={wechatVerificationCode}
            onChange={setWechatVerificationCode}
          />
        </Form>
      </Modal>
    </>
  );
};

export default OAuthButtons;
