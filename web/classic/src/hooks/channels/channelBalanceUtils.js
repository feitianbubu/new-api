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

import { getQuotaPerUnit } from '../../helpers/quota';

export const channelLiveBalanceUsd = (channel) => {
  const balance = channel?.balance || 0;
  const snapshot = channel?.balance_snapshot;
  if (snapshot == null) return balance;
  return balance - ((channel.used_quota || 0) - snapshot) / getQuotaPerUnit();
};

export const estimateChannelDaysRemaining = (liveBalance, usage) => {
  if (!usage || usage.quota <= 0 || liveBalance <= 0) return null;
  const avgDailyUsd =
    usage.quota / Math.max(usage.active_days, 1) / getQuotaPerUnit();
  return liveBalance / avgDailyUsd;
};
