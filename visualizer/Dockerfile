# Use an official Python runtime as a parent image
FROM python:3.10

# Set the working directory to /apps
WORKDIR /app

# Copy the current directory contents into the container at /apps
COPY . /app

# Install any needed packages specified in requirements.txt
RUN pip install -r requirements.txt -i https://pypi.tuna.tsinghua.edu.cn/simple

# Run apps.py when the container launches
CMD ["python", "process.py"]