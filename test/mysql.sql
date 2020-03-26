
DROP TABLE IF EXISTS `t_demo`;
CREATE TABLE `t_demo`  (
  `id` varchar(50)  NOT NULL COMMENT ' ',
  `userName` varchar(30)  NOT NULL COMMENT '姓名',
  `password` varchar(50)  NOT NULL COMMENT '密码',
  `mobile` varchar(16)  NOT NULL COMMENT '手机号码',
  `createTime` datetime(0) NOT NULL DEFAULT CURRENT_TIMESTAMP(0),
  `active` int(0) NOT NULL DEFAULT 1 COMMENT '是否有效(0否,1是)',
  PRIMARY KEY (`id`)
) ENGINE = InnoDB CHARACTER SET = utf8mb4  COMMENT = '栗子' ;
