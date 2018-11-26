import React from 'react';
import PropTypes from 'prop-types';
import { Paper } from '@material-ui/core';
import { withStyles } from '@material-ui/core/styles';
import { pageStyles } from '../styles';

const styles = theme => ({
  ...pageStyles(theme)
});

const About = props => {
  const { classes } = props;
  return (
    <div className={classes.root}>
      <Paper className={classes.paper}>
        <p>
          A cryptocurrency trading bot supporting multiple exchanges written in
          Golang. Join our slack to discuss all things related to
          GoCryptoTrader!{' '}
          <a href="https://gocryptotrader.herokuapp.com/">
            GoCryptoTrader Slack
          </a>
        </p>
      </Paper>
    </div>
  );
};

About.propTypes = {
  classes: PropTypes.object.isRequired,
  theme: PropTypes.object.isRequired
};

export default withStyles(styles, { withTheme: true })(About);
